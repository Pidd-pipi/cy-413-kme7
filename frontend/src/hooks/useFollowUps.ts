import {useCallback, useEffect, useState,useSyncExternalStore} from 'react';
import {message} from 'antd';
import {listFollowUps, respondFollowUp} from '../api/followUp';
import {followUpStore} from '../stores/followUpStore';
import type {FollowUp, FollowUpResult, FollowUpStatus} from '../types';

// 回访数据统一入口：加载后写入共享 store，花园/情绪记录/日记页/顶部角标读取一致状态。
export function useFollowUps(){
  const followUps=useSyncExternalStore(followUpStore.subscribe,followUpStore.get);
  const [loading,setLoading]=useState(false);

  const load=useCallback(()=>{
    setLoading(true);
    return listFollowUps().then(followUpStore.set).catch(e=>message.error(e.message)).finally(()=>setLoading(false));
  },[]);

  useEffect(()=>{load()},[load]);

  const respond=useCallback(async(id:number,result:FollowUpResult)=>{
    const kept=await respondFollowUp(id,result); // 重复/并发时后端返回唯一那条
    followUpStore.upsert(kept);
    return kept;
  },[]);

  const byDate=useCallback((date:string)=>followUps.find(f=>f.trigger_date.slice(0,10)===date.slice(0,10)),[followUps]);
  const pendingForToday=useCallback(()=>{
    const today=new Date().toISOString().slice(0,10);
    return followUps.filter(f=>f.status==='pending'&&f.scheduled_date.slice(0,10)<=today);
  },[followUps]);

  return {followUps,loading,load,respond,byDate,pendingForToday};
}

export type {FollowUpStatus};
