const requiredVariables = [
  'MAESTRO_E2E_API_URL',
  'MAESTRO_E2E_USERNAME',
  'MAESTRO_E2E_EMAIL',
  'MAESTRO_E2E_PASSWORD',
  'MAESTRO_E2E_KNOWLEDGE_NAME',
];

for (const name of requiredVariables) {
  if (!process.env[name]?.trim()) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
}

const apiURL = process.env.MAESTRO_E2E_API_URL.replace(/\/+$/, '');
const username = process.env.MAESTRO_E2E_USERNAME.trim();

const registrationResponse = await fetch(`${apiURL}/api/auth/register`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username,
    email: process.env.MAESTRO_E2E_EMAIL.trim(),
    password: process.env.MAESTRO_E2E_PASSWORD,
  }),
});
const registrationEnvelope = await registrationResponse.json().catch(() => null);

if (!registrationResponse.ok || !registrationEnvelope || registrationEnvelope.code !== 0) {
  const message = registrationEnvelope?.message || `HTTP ${registrationResponse.status}`;
  throw new Error(`/api/auth/register failed: ${message}`);
}

const accessToken = registrationEnvelope.data?.access_token;
if (!accessToken || registrationEnvelope.data.username !== username) {
  throw new Error('registration did not return the expected disposable user session');
}

const listResponse = await fetch(`${apiURL}/api/knowledge-base/`, {
  headers: { Authorization: `Bearer ${accessToken}` },
});
const listEnvelope = await listResponse.json().catch(() => null);
const initialItems = listEnvelope?.data?.list;

if (!listResponse.ok || !listEnvelope || listEnvelope.code !== 0 || !Array.isArray(initialItems)) {
  const message = listEnvelope?.message || `HTTP ${listResponse.status}`;
  throw new Error(`/api/knowledge-base/ failed: ${message}`);
}

if (initialItems.length !== 1 || !initialItems[0]?.is_default) {
  throw new Error(`expected one initial default knowledge base, received ${initialItems.length}`);
}

const knowledgeName = process.env.MAESTRO_E2E_KNOWLEDGE_NAME.trim();
if (initialItems.some((item) => item?.name === knowledgeName)) {
  throw new Error('synthetic knowledge base name already exists before the App flow');
}

console.log(`Prepared disposable knowledge fixture for ${username}.`);
