import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  Activity, BookOpen, Check, ChevronRight, Clock3, Flame, Gamepad2,
  LayoutDashboard, Library, Moon, Plus, RefreshCw, Send, Sparkles, Trophy,
  X
} from 'lucide-react';
import './styles.css';

const API = '/api';
const emptyDashboard = {
  radar_scores: { study_score: 64, sport_score: 42, hobby_score: 78, routine_score: 71 },
  today_study_minutes: 0,
  today_sport_done: false,
  today_sleep_hours: null,
  active_streaks: [],
  upcoming_f1: [],
  recent_media: []
};

function apiFetch(path, options = {}) {
  const token = localStorage.getItem('pixellife_token');
  return fetch(`${API}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers }
  }).then(async (response) => {
    if (!response.ok) throw new Error(await response.text());
    return response.status === 204 ? null : response.json();
  });
}

function formatTime(seconds) {
  const minutes = Math.floor(seconds / 60).toString().padStart(2, '0');
  const rest = (seconds % 60).toString().padStart(2, '0');
  return `${minutes}:${rest}`;
}

function Radar({ values }) {
  const points = [values.study_score, values.sport_score, values.hobby_score, values.routine_score];
  const coords = points.map((value, index) => {
    const angle = (-90 + index * 90) * Math.PI / 180;
    const radius = 54 * Math.max(value, 8) / 100;
    return `${50 + Math.cos(angle) * radius},${50 + Math.sin(angle) * radius}`;
  }).join(' ');
  return <div className="radar-wrap">
    <svg viewBox="0 0 100 100" className="radar" role="img" aria-label="Баланс сфер жизни">
      {[18, 36, 54].map((radius) => <polygon key={radius} points={`50,${50 - radius} ${50 + radius},50 50,${50 + radius} ${50 - radius},50`} className="radar-grid" />)}
      <line x1="50" y1="0" x2="50" y2="100" className="radar-axis" /><line x1="0" y1="50" x2="100" y2="50" className="radar-axis" />
      <polygon points={coords} className="radar-fill" /><polyline points={coords} className="radar-line" />
    </svg>
    <div className="radar-label top">Учёба <b>{Math.round(points[0])}</b></div>
    <div className="radar-label right">Спорт <b>{Math.round(points[1])}</b></div>
    <div className="radar-label bottom">Рутина <b>{Math.round(points[3])}</b></div>
    <div className="radar-label left">Хобби <b>{Math.round(points[2])}</b></div>
  </div>;
}

function App() {
  const [dashboard, setDashboard] = useState(() => JSON.parse(localStorage.getItem('pixellife_dashboard') || 'null') || emptyDashboard);
  const [view, setView] = useState('home');
  const [toast, setToast] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const [pomodoro, setPomodoro] = useState(() => JSON.parse(localStorage.getItem('pixellife_pomodoro') || 'null') || { seconds: 25 * 60, running: false, category: 'SAT' });

  useEffect(() => {
    localStorage.setItem('pixellife_pomodoro', JSON.stringify(pomodoro));
    if (!pomodoro.running) return undefined;
    const tick = setInterval(() => setPomodoro((current) => {
      if (current.seconds <= 1) {
        const payload = { category: current.category, duration_minutes: 25, quality_rating: 4, notes: 'Pomodoro' };
        apiFetch('/study/sessions', { method: 'POST', body: JSON.stringify(payload) }).catch(() => undefined);
        setDashboard((dashboardState) => ({ ...dashboardState, today_study_minutes: dashboardState.today_study_minutes + 25 }));
        setToast('Помодоро завершён. +25 минут в журнале.');
        return { ...current, seconds: 25 * 60, running: false };
      }
      return { ...current, seconds: current.seconds - 1 };
    }), 1000);
    return () => clearInterval(tick);
  }, [pomodoro]);

  useEffect(() => {
    apiFetch('/dashboard').then(setDashboard).catch(() => setToast('Офлайн-режим: показываю последний снимок'));
  }, []);

  useEffect(() => {
    localStorage.setItem('pixellife_dashboard', JSON.stringify(dashboard));
  }, [dashboard]);

  useEffect(() => { if (toast) { const timeout = setTimeout(() => setToast(''), 3200); return () => clearTimeout(timeout); } }, [toast]);
  useEffect(() => {
    if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => undefined);
  }, []);

  const addStudy = async () => {
    const payload = { category: pomodoro.category, duration_minutes: 25, quality_rating: 4, notes: 'Pomodoro' };
    try { await apiFetch('/study/sessions', { method: 'POST', body: JSON.stringify(payload) }); } catch { /* local-first UX */ }
    setDashboard((current) => ({ ...current, today_study_minutes: current.today_study_minutes + 25 }));
    setToast('+25 минут в журнале');
  };

  const completeSport = async () => {
    try { await apiFetch('/sport/sessions', { method: 'POST', body: JSON.stringify({ workout_type: 'Домашняя тренировка', completed: true }) }); } catch { /* local-first UX */ }
    setDashboard((current) => ({ ...current, today_sport_done: true }));
    setToast('Тренировка отмечена');
  };

  return <div className="app-shell">
    <header className="topbar">
      <div className="brand"><div className="brand-mark">PL</div><div><strong>PIXELLIFE</strong><span>личный ритм-трекер</span></div></div>
      <div className="header-actions"><span className="sync-dot" title="Синхронизация активна" /><button className="icon-button" onClick={() => apiFetch('/dashboard').then(setDashboard).catch(() => setToast('Сервер пока недоступен'))} aria-label="Обновить"><RefreshCw size={17} /></button></div>
    </header>
    <main className="content">
      {view === 'home' && <>
        <section className="hero-row"><div><p className="eyebrow">ВТОРНИК · 22 СЕНТЯБРЯ</p><h1>Твой день,<br /><em>твой ритм.</em></h1><p className="hero-copy">Маленькие действия складываются в большую статистику.</p></div><div className="level-badge"><Trophy size={17} /><span>LVL 07</span><b>1 240 XP</b></div></section>
        <section className="dashboard-grid">
          <article className="panel balance-panel"><div className="panel-heading"><div><span className="kicker">СЕГОДНЯ</span><h2>Баланс жизни</h2></div><span className="status-pill">В ФОКУСЕ</span></div><Radar values={dashboard.radar_scores} /></article>
          <article className="panel focus-panel"><div className="panel-heading"><div><span className="kicker">ФОКУС-СЕССИЯ</span><h2>Pomodoro</h2></div><Clock3 size={19} /></div><div className="timer">{formatTime(pomodoro.seconds)}</div><div className="timer-meta"><select value={pomodoro.category} onChange={(event) => setPomodoro({ ...pomodoro, category: event.target.value })}><option>SAT</option><option>IELTS</option><option>NISH</option></select><span>25 минут</span></div><div className="timer-actions"><button className="primary-button" onClick={() => setPomodoro({ ...pomodoro, running: !pomodoro.running })}>{pomodoro.running ? 'Пауза' : 'Старт'} <ChevronRight size={15} /></button><button className="ghost-button" onClick={() => setPomodoro({ ...pomodoro, seconds: 25 * 60, running: false })}>Сброс</button></div></article>
        </section>
        <section className="section-heading"><div><span className="kicker">БЫСТРАЯ СВОДКА</span><h2>Сегодня</h2></div><button className="text-button" onClick={() => setShowAdd(true)}>Добавить <Plus size={15} /></button></section>
        <section className="stats-grid"><Stat icon={<BookOpen />} label="Учёба" value={`${dashboard.today_study_minutes} мин`} accent="red" /><Stat icon={<Activity />} label="Спорт" value={dashboard.today_sport_done ? 'Сделано' : 'Не начато'} accent="blue" action={!dashboard.today_sport_done ? completeSport : undefined} /><Stat icon={<Moon />} label="Сон" value={dashboard.today_sleep_hours ? `${dashboard.today_sleep_hours} ч` : 'Нет записи'} accent="yellow" /></section>
        <section className="section-heading"><div><span className="kicker">ПРОДОЛЖИТЬ</span><h2>Твои линии</h2></div></section>
        <section className="lines-grid"><LineCard icon={<Flame />} title="Гитара" meta="Хобби · стрик" value={dashboard.active_streaks[0]?.current_streak || 0} suffix="дней" accent="orange" /><LineCard icon={<Library />} title="Полка" meta="Медиа · в процессе" value={dashboard.recent_media.length || 0} suffix="тайтлов" accent="green" /><LineCard icon={<Gamepad2 />} title="F1 weekend" meta="Следующее событие" value={dashboard.upcoming_f1[0]?.name || 'Добавить дату'} suffix="" accent="violet" /></section>
      </>}
      {view !== 'home' && <PlaceholderView view={view} onBack={() => setView('home')} />}
    </main>
    <nav className="bottom-nav">{[[LayoutDashboard, 'home', 'Обзор'], [BookOpen, 'study', 'Учёба'], [Library, 'shelf', 'Полка'], [Activity, 'habits', 'Ритм']].map(([Icon, key, label]) => <button key={key} className={view === key ? 'active' : ''} onClick={() => setView(key)}><Icon size={19} /><span>{label}</span></button>)}</nav>
    {showAdd && <QuickAdd onClose={() => setShowAdd(false)} onStudy={addStudy} onSport={completeSport} />}
    {toast && <div className="toast"><Check size={16} />{toast}</div>}
  </div>;
}

function Stat({ icon, label, value, accent, action }) { return <article className={`stat-card ${accent}`}><div className="stat-icon">{icon}</div><div><span>{label}</span><strong>{value}</strong></div>{action && <button className="mini-check" onClick={action} aria-label="Отметить"><Check size={15} /></button>}</article>; }
function LineCard({ icon, title, meta, value, suffix, accent }) { return <article className={`line-card ${accent}`}><div className="line-icon">{icon}</div><div><span>{meta}</span><h3>{title}</h3><strong>{value} <small>{suffix}</small></strong></div><ChevronRight size={17} className="line-arrow" /></article>; }
function QuickAdd({ onClose, onStudy, onSport }) { return <div className="modal-backdrop" onClick={onClose}><div className="quick-modal" onClick={(event) => event.stopPropagation()}><button className="modal-close" onClick={onClose}><X size={17} /></button><span className="kicker">БЫСТРАЯ ОТМЕТКА</span><h2>Что сделал сегодня?</h2><div className="quick-options"><button onClick={() => { onStudy(); onClose(); }}><BookOpen size={20} /><span>Учился</span><small>+25 минут</small></button><button onClick={() => { onSport(); onClose(); }}><Activity size={20} /><span>Тренировался</span><small>домашняя тренировка</small></button><button onClick={onClose}><Moon size={20} /><span>Записать сон</span><small>откроется форма</small></button></div></div></div>; }
function PlaceholderView({ view, onBack }) { const titles = { study: ['Учёба', 'Трекай время и результат, а не только часы.'], shelf: ['Полка', 'Книги и сериалы в одном спокойном месте.'], habits: ['Ритм', 'Сон, спорт и хобби без лишнего шума.'] }; const [title, copy] = titles[view]; return <section className="placeholder"><span className="kicker">РАЗДЕЛ</span><h1>{title}</h1><p>{copy}</p><button className="primary-button" onClick={onBack}>Вернуться к обзору <ChevronRight size={15} /></button></section>; }

createRoot(document.getElementById('root')).render(<App />);
