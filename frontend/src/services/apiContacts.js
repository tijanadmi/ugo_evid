const contactPath = id => `/ugo_dob_lica/${id}`;
export const getContact = (api, id) => api(contactPath(id)).then(result => result.data);
export const acquireContact = (api, id) => api(`${contactPath(id)}/lock`, { method: 'POST' });
export const renewContact = (api, id, token) => api(`${contactPath(id)}/lock`, { method: 'PUT', body: { lock_token: token } });
export const releaseContact = (api, id, token) => api(`${contactPath(id)}/lock`, { method: 'DELETE', body: { lock_token: token } });
export const saveContact = (api, id, body) => api(id ? contactPath(id) : '/ugo_dob_lica', { method: id ? 'PUT' : 'POST', body });
export const deleteContact = (api, id, version, token) => api(contactPath(id), { method: 'DELETE', body: { version, lock_token: token } });
export async function getContactRoles(api) {
  const roles = [];
  for (let page = 1; ; page++) {
    const result = await api(`/ugo_dob_lica_rola?page_id=${page}&page_size=200`);
    roles.push(...(result.items || []));
    if (!result.items?.length || roles.length >= result.total) return roles;
  }
}
