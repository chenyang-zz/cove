package tool

// CloneDescriptor 返回工具描述的独立副本。
//
// 该函数会复制 schema、required、annotations 和 strict 指针，调用方后续修改原始
// descriptor 不会影响副本。嵌套的任意值只做浅拷贝，适合当前 JSON schema 投影使用。
func CloneDescriptor(descriptor Descriptor) Descriptor {
	descriptor.Schema = cloneSchema(descriptor.Schema)
	descriptor.Annotations = cloneMap(descriptor.Annotations)
	return descriptor
}

// CloneDescriptors 返回工具描述列表的独立副本。
//
// nil 输入会返回 nil；非 nil 输入会返回新的切片，并逐个调用 CloneDescriptor 复制元素。
func CloneDescriptors(descriptors []Descriptor) []Descriptor {
	if descriptors == nil {
		return nil
	}
	cloned := make([]Descriptor, len(descriptors))
	for i, descriptor := range descriptors {
		cloned[i] = CloneDescriptor(descriptor)
	}
	return cloned
}

func cloneDescriptor(descriptor Descriptor) Descriptor {
	return CloneDescriptor(descriptor)
}

func cloneSchema(schema Schema) Schema {
	schema.Parameters.Properties = cloneProperties(schema.Parameters.Properties)
	schema.Parameters.Required = cloneStrings(schema.Parameters.Required)
	schema.Parameters.AdditionalProperties = cloneAny(schema.Parameters.AdditionalProperties)
	if schema.Strict != nil {
		strict := *schema.Strict
		schema.Strict = &strict
	}
	return schema
}

func cloneProperties(values map[string]PropertySchema) map[string]PropertySchema {
	if values == nil {
		return nil
	}
	cloned := make(map[string]PropertySchema, len(values))
	for key, value := range values {
		cloned[key] = PropertySchema(cloneMap(value))
	}
	return cloned
}

func cloneSetDescriptor(descriptor SetDescriptor) SetDescriptor {
	descriptor.Tags = cloneStrings(descriptor.Tags)
	descriptor.Annotations = cloneMap(descriptor.Annotations)
	return descriptor
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []string:
		return cloneStrings(typed)
	case []any:
		cloned := make([]any, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return value
	}
}
