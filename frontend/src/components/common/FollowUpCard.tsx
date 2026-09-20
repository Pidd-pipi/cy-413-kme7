import {Button, Card, Popconfirm, Space, Tag, Typography} from 'antd';
import dayjs from 'dayjs';
import {Link} from 'react-router-dom';
import {FOLLOWUP_RESULT_LABELS, FOLLOWUP_SOURCE_LABELS, FOLLOWUP_STATUS_LABELS} from '../../constants/followUp';
import type {FollowUp, FollowUpResult} from '../../types';

const STATUS_COLORS:Record<string,string>={pending:'orange',responded:'green',revoked:'default'};
const RESULT_COLORS:Record<string,string>={better:'green',struggling:'volcano'};

interface Props{
  followUp:FollowUp;
  onRespond:(id:number,result:FollowUpResult)=>Promise<unknown>;
  compact?:boolean;
}

// 回访只能由本人确认好转或仍困扰；次日到期前只作提示，已提交结果不允许再改。
export function FollowUpCard({followUp,onRespond,compact}:Props){
  const triggerDay=followUp.trigger_date.slice(0,10);
  const due=dayjs(followUp.scheduled_date).isSame(dayjs(),'day')||dayjs(followUp.scheduled_date).isBefore(dayjs(),'day');
  return (
    <Card size="small" className="followup-card" data-status={followUp.status}>
      <div className="followup-card__top">
        <b>🫧 次日回访 · {triggerDay}</b>
        <Tag color={STATUS_COLORS[followUp.status]}>{FOLLOWUP_STATUS_LABELS[followUp.status]}</Tag>
      </div>
      <div className="muted">首次触发：{FOLLOWUP_SOURCE_LABELS[followUp.source]}</div>
      {followUp.status==='pending' ? (
        !due ? (
          <Typography.Text type="secondary" style={{display:'inline-block',marginTop:8}}>
            将于 {followUp.scheduled_date.slice(0,10)} 开放回访
          </Typography.Text>
        ) : compact ? (
          <Link to="/moods" style={{display:'inline-block',marginTop:8,color:'#d48806'}}>今天感觉怎么样？去回应回访 →</Link>
        ) : (
          <Space wrap style={{marginTop:8}}>
            <Popconfirm title="确认今天已经好转了吗？" okText="确认好转" cancelText="再想想" onConfirm={()=>onRespond(followUp.id,'better')}>
              <Button type="primary">我好转了</Button>
            </Popconfirm>
            <Popconfirm title="仍然被低落情绪困扰着吗？" okText="仍困扰" cancelText="取消" onConfirm={()=>onRespond(followUp.id,'struggling')}>
              <Button danger>仍在困扰</Button>
            </Popconfirm>
          </Space>
        )
      ) : followUp.status==='responded' ? (
        <div style={{marginTop:8}}>
          <Tag color={RESULT_COLORS[followUp.result]}>本人确认：{FOLLOWUP_RESULT_LABELS[followUp.result as FollowUpResult]}</Tag>
          {followUp.responded_at && <span className="muted">{followUp.responded_at.slice(0,10)}</span>}
        </div>
      ) : (
        <Typography.Text type="secondary" style={{marginTop:8,display:'inline-block'}}>情绪指数已回升，回访已自动撤销</Typography.Text>
      )}
    </Card>
  );
}
