export function getUnregisteredContracts(api, page, filter, signal) {
  return api(`/sapugovori/neevidentirani?${new URLSearchParams({ page_id: page, page_size: 20, filter })}`, { signal });
}
export function prepareContract(api, id, signal) {
  return api(`/sapugovori/${id}/priprema`, { signal });
}
export function registerContract(api, body) {
  return api('/ugo_evid', { method: 'POST', body });
}
