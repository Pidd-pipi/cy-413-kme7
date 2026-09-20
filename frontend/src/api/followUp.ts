import {request} from '../utils/request';
import type {FollowUp, FollowUpResult, FollowUpStatus} from '../types';

export const listFollowUps=(params?:{status?:FollowUpStatus;trigger_date?:string})=>{
  const qs=new URLSearchParams();
  if(params?.status) qs.set('status',params.status);
  if(params?.trigger_date) qs.set('trigger_date',params.trigger_date);
  const suffix=qs.toString()?`?${qs.toString()}`:'';
  return request<FollowUp[]>(`/follow-ups${suffix}`);
};
// 只能由本人确认好转或仍困扰；重复及并发提交后端只保留一条，这里始终返回唯一那条
export const respondFollowUp=(id:number,result:FollowUpResult)=>request<FollowUp>(`/follow-ups/${id}/respond`,{method:'POST',body:JSON.stringify({result})});
