package neo4j

import (
	"context"
	"strings"

	"github.com/boxify/api-go/internal/xerr"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

type Row map[string]any

type Tx interface {
	Run(ctx context.Context, cypher string, params map[string]any) ([]Row, error)
}

type Client struct {
	driver   neo4jdriver.Driver
	database string
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	driver, err := neo4jdriver.NewDriver(
		strings.TrimSpace(cfg.URI),
		neo4jdriver.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, xerr.Wrapf(err, "创建 Neo4j driver 失败")
	}
	client := &Client{driver: driver, database: normalizeDatabase(cfg.Database)}
	if err := client.Verify(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, err
	}
	return client, nil
}

func (c *Client) Verify(ctx context.Context) error {
	return c.Ping(ctx)
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.driver == nil {
		return xerr.BadRequest("Neo4j 客户端未初始化")
	}
	if err := c.driver.VerifyConnectivity(ctx); err != nil {
		return xerr.Wrapf(err, "验证 Neo4j 连接失败")
	}
	return nil
}

func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.driver == nil {
		return nil
	}
	if err := c.driver.Close(ctx); err != nil {
		return xerr.Wrapf(err, "关闭 Neo4j 连接失败")
	}
	return nil
}

func (c *Client) Read(ctx context.Context, cypher string, params map[string]any) ([]Row, error) {
	return c.query(ctx, cypher, params, neo4jdriver.EagerResultTransformer, neo4jdriver.ExecuteQueryWithReadersRouting())
}

func (c *Client) Write(ctx context.Context, cypher string, params map[string]any) ([]Row, error) {
	return c.query(ctx, cypher, params, neo4jdriver.EagerResultTransformer, neo4jdriver.ExecuteQueryWithWritersRouting())
}

func (c *Client) ReadTx(ctx context.Context, fn func(tx Tx) error) error {
	return c.runTx(ctx, neo4jdriver.AccessModeRead, fn)
}

func (c *Client) WriteTx(ctx context.Context, fn func(tx Tx) error) error {
	return c.runTx(ctx, neo4jdriver.AccessModeWrite, fn)
}

func (c *Client) query(ctx context.Context, cypher string, params map[string]any, transformer func() neo4jdriver.ResultTransformer[*neo4jdriver.EagerResult], opts ...neo4jdriver.ExecuteQueryConfigurationOption) ([]Row, error) {
	queryOpts := append([]neo4jdriver.ExecuteQueryConfigurationOption{}, opts...)
	if c.database != "" {
		queryOpts = append(queryOpts, neo4jdriver.ExecuteQueryWithDatabase(c.database))
	}
	result, err := neo4jdriver.ExecuteQuery[*neo4jdriver.EagerResult](ctx, c.driver, cypher, normalizeParams(params), transformer, queryOpts...)
	if err != nil {
		return nil, xerr.Wrapf(err, "执行 Neo4j 查询失败")
	}
	return rowsFromRecords(result.Records), nil
}

func (c *Client) runTx(ctx context.Context, mode neo4jdriver.AccessMode, fn func(tx Tx) error) error {
	if fn == nil {
		return xerr.BadRequest("Neo4j 事务函数不能为空")
	}
	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode:   mode,
		DatabaseName: c.database,
	})
	defer func() {
		_ = session.Close(ctx)
	}()

	work := func(tx neo4jdriver.ManagedTransaction) (any, error) {
		return nil, fn(managedTx{tx: tx})
	}
	var err error
	if mode == neo4jdriver.AccessModeRead {
		_, err = session.ExecuteRead(ctx, work)
	} else {
		_, err = session.ExecuteWrite(ctx, work)
	}
	if err != nil {
		return xerr.Wrapf(err, "执行 Neo4j 事务失败")
	}
	return nil
}

type managedTx struct {
	tx neo4jdriver.ManagedTransaction
}

func (t managedTx) Run(ctx context.Context, cypher string, params map[string]any) ([]Row, error) {
	result, err := t.tx.Run(ctx, cypher, normalizeParams(params))
	if err != nil {
		return nil, xerr.Wrapf(err, "执行 Neo4j 事务查询失败")
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, xerr.Wrapf(err, "读取 Neo4j 事务查询结果失败")
	}
	return rowsFromRecords(records), nil
}

func normalizeParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	return params
}

func normalizeDatabase(database string) string {
	return strings.TrimSpace(database)
}

func rowsFromRecords(records []*neo4jdriver.Record) []Row {
	rows := make([]Row, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		row := Row{}
		for i, key := range record.Keys {
			if i < len(record.Values) {
				row[key] = record.Values[i]
			}
		}
		rows = append(rows, row)
	}
	return rows
}
