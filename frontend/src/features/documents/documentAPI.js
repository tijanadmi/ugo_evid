export const documentSessionService = {
 acquireContact: (api, id) => api(`/ugo_doc/${id}/lock`, { method: 'POST' }),
 renewContact: (api, id, token) => api(`/ugo_doc/${id}/lock`, { method: 'PUT', body: { lock_token: token } }),
 releaseContact: (api, id, token) => api(`/ugo_doc/${id}/lock`, { method: 'DELETE', body: { lock_token: token } }),
 getContact: (api, id) => api(`/ugo_doc/${id}`),
};
export async function uploadDocument(api, id, metadata, file) {
 const body = new FormData();
 body.append('metadata', JSON.stringify(metadata));
 body.append('file', file);
 return api(`/ugo_evid/${id}/dokumenti`, { method: 'POST', body });
}
