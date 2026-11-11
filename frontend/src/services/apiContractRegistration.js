export function getUnregisteredContracts(api, page, filter, signal, orgID) {
  return api(
    `/sapugovori/neevidentirani?${new URLSearchParams({ page_id: page, page_size: 20, filter, id_ugo_org: orgID })}`,
    { signal },
  );
}
export function prepareContract(api, id, signal, orgID) {
  return api(`/sapugovori/${id}/priprema?id_ugo_org=${orgID}`, { signal });
}
export function registerContract(api, body, orgID) {
  return api(`/ugo_evid?id_ugo_org=${orgID}`, { method: "POST", body });
}
