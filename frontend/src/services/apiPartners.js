export function getMyPartners(api, page, pageSize, naziv, signal, org = 0) {
  return api(
    `/moji_partneri?${new URLSearchParams({ page_id: page, page_size: pageSize, naziv, ...(org ? { id_ugo_org: org } : {}) })}`,
    { signal },
  );
}
