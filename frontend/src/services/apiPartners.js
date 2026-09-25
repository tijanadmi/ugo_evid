export function getMyPartners(api, page, pageSize, naziv, signal) {
  return api(`/moji_partneri?${new URLSearchParams({ page_id: page, page_size: pageSize, naziv })}`, { signal });
}
