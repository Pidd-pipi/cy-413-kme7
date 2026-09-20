import {Badge, Tooltip} from 'antd';
import {Link} from 'react-router-dom';
import {useEffect} from 'react';
import {useFollowUps} from '../../hooks/useFollowUps';
import dayjs from 'dayjs';

// 全局待回访角标：任意页面都能看到已到期（次日及以后未回应）的低落回访数量。
export function FollowUpBadge(){
  const {followUps,load}=useFollowUps();
  useEffect(()=>{
    const id=setInterval(load,60_000); // 每分钟轻量同步一次，保证跨页停留也不过期
    return ()=>clearInterval(id);
  },[load]);
  const due=followUps.filter(f=>f.status==='pending'&&!dayjs(f.scheduled_date).isAfter(dayjs(),'day'));
  return (
    <Tooltip title={due.length?`有 ${due.length} 条低落情绪回访等待你回应`:'暂无待回应的回访'}>
      <Link to="/moods" style={{fontSize:18,lineHeight:1}}>
        <Badge count={due.length} size="small" offset={[-2,2]}><span>🫧</span></Badge>
      </Link>
    </Tooltip>
  );
}
