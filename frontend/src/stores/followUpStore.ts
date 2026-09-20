import type {FollowUp} from '../types';

let followUps:FollowUp[]=[];
const listeners=new Set<()=>void>();
function emit(){listeners.forEach(l=>l());}

// 心情花园、情绪记录、日记页与顶部角标共享同一份回访快照；
// 每次从服务端刷新后整体替换，各处读取到的状态始终一致（含刷新页面后）。
export const followUpStore={
  get:()=>followUps,
  set:(values:FollowUp[])=>{followUps=values;emit();},
  upsert:(v:FollowUp)=>{
    const idx=followUps.findIndex(f=>f.id===v.id);
    followUps=idx>=0?[...followUps.slice(0,idx),v,...followUps.slice(idx+1)]:[v,...followUps];
    emit();
  },
  subscribe:(l:()=>void)=>{listeners.add(l);return ()=>{listeners.delete(l);}},
};
