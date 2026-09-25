export function getMyPartners(api, page, pageSize, signal) {
  return api(`/moji_partneri?${new URLSearchParams({ page_id: page, page_size: pageSize })}`, { signal });
}
