export const maintenanceService = {
 acquireContact: (api,id) => api(`/ugo_evid/${id}/odrzavanje/lock`,{method:'POST'}),
 renewContact: (api,id,token) => api(`/ugo_evid/${id}/odrzavanje/lock`,{method:'PUT',body:{lock_token:token}}),
 releaseContact: (api,id,token) => api(`/ugo_evid/${id}/odrzavanje/lock`,{method:'DELETE',body:{lock_token:token}}),
 getContact: async (api,id) => {
  const data=await api(`/ugo_evid/${id}/podesavanja`);
  if(!data.settings.odrzavanje) throw new Error('Odrzavanje nije pronadjeno.');
  return {...data.settings.odrzavanje,id,settings:data.settings};
 },
};
