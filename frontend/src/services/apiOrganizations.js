export async function getOrganizations(api, signal) {
  const all = [];
  let page = 1;
  while (!signal?.aborted) {
    const result = await api(`/ugo_org?page_id=${page}&page_size=100`, { signal });
    const items = result.items || [];
    all.push(...items);
    if (items.length === 0 || all.length >= result.total) break;
    page++;
  }
  return all;
}
