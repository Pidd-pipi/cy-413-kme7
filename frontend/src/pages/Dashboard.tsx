import {Button,Card,Col,Form,Input,Row,Slider,Typography,message} from 'antd';import {useEffect,useState} from 'react';import {listMoods,saveMood} from '../api/mood';import {listAssessments} from '../api/assessment';import {MoodSelector} from '../components/common/MoodSelector';import {MoodTrendChart} from '../components/common/MoodTrendChart';import {CheckInReminder} from '../components/common/CheckInReminder';import {AssessmentCard} from '../components/common/AssessmentCard';import {useMoodStats} from '../hooks/useMoodStats';import {useCheckIns} from '../hooks/useCheckIns';import type {Assessment,Mood,MoodTag} from '../types';import {useNavigate} from 'react-router-dom';import dayjs from 'dayjs';
export function Dashboard(){
  const [moods,setMoods]=useState<Mood[]>([]),[assessments,setAssessments]=useState<Assessment[]>([]),[tags,setTags]=useState<MoodTag[]>(['calm']);
  // 所有到期/逾期仍待确认的回访；与情绪记录、日记本共用同一接口，刷新后状态一致。
  const {duePending:dueCheckIns,submitting:checkInSubmitting,load:loadCheckIns,respond:respondCheckIn}=useCheckIns({status:'pending'});
  const nav=useNavigate();
  const weekStart=dayjs().subtract(6,'day').startOf('day');const weekEnd=dayjs().endOf('day');
  const weekMoods=moods.filter(m=>{const d=dayjs(m.record_date);return !d.isBefore(weekStart)&&!d.isAfter(weekEnd)});
  const stats=useMoodStats(weekMoods);
  const load=()=>Promise.all([listMoods(),listAssessments(),loadCheckIns()]).then(([m,a])=>{setMoods(m);setAssessments(a)}).catch(e=>message.error(e.message));
  useEffect(()=>{load()},[]);
  const handleRespond=async(id:number,result:'improved'|'still_troubled')=>{const r=await respondCheckIn(id,result);if(r.ok){message.success('回访结果已记录，谢谢你的温柔确认');load()}else{message.error(r.error.message||'回访已有结果，只保留第一条')}};
  return <><CheckInReminder pending={dueCheckIns} submitting={checkInSubmitting} onRespond={handleRespond}/>
  <section className="hero"><div><Typography.Title level={1}>今天，花园感觉如何？</Typography.Title><p>记录不是评判，而是温柔地看见自己。</p></div><div className="stat">本周平均心情<br/><b>{stats.average||'—'}</b><small>/10</small></div></section>
  <Row gutter={[20,20]}><Col xs={24} lg={15}><Card title="本周情绪曲线"><MoodTrendChart moods={weekMoods}/></Card></Col>
  <Col xs={24} lg={9}><Card title="快速记录情绪"><Form layout="vertical" onFinish={async v=>{try{await saveMood({mood_level:v.level,mood_tags:tags,note:v.note||'',record_date:new Date().toISOString().slice(0,10),source:'dashboard'});message.success(v.level<=3?'情绪已记录，明天会来回访你':'情绪已记录');load()}catch(e){message.error((e as Error).message)}}} initialValues={{level:7}}><Form.Item label="心情指数" name="level"><Slider min={1} max={10}/></Form.Item><Form.Item label="此刻的感受"><MoodSelector value={tags} onChange={setTags}/></Form.Item><Form.Item label="想说的话" name="note"><Input.TextArea rows={2} placeholder="写下一句就很好"/></Form.Item><Button type="primary" htmlType="submit">种下一朵心情花</Button></Form></Card></Col>
  <Col span={24}><Typography.Title level={3}>推荐给你的片刻关照</Typography.Title><Row gutter={[16,16]}>{assessments.slice(0,2).map(a=><Col xs={24} md={12} key={a.id}><AssessmentCard assessment={a} onTake={()=>nav('/assessments')}/></Col>)}</Row></Col></Row></>}
