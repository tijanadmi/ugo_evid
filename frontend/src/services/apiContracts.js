export function getContract(api, id, signal) {
  return api(`/ugo_evid/${id}/detalji`, { signal });
}

export function getContracts(api, { status, page, pageSize, orgID, signal }) {
  const query = new URLSearchParams({ page_id: page, page_size: pageSize, id_ugo_org: orgID });
  return api(`/ugo_evid/${status}?${query}`, { signal });
}
