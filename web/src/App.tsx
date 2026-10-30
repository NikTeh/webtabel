import { useState, useEffect } from 'react';
import { Login } from './Login';
import { UsersTab } from './UsersTab';
import { MerchandisersTab } from './MerchandisersTab';
import { StoresTab } from './StoresTab';
import { ManufacturersTab } from './ManufacturersTab';
import { MatricesTab } from './MatricesTab';
import { QuestionnairesTab } from './QuestionnairesTab';
import { InfoTab } from './InfoTab';
import { ReportsTab } from './ReportsTab';

// Импортируем вкладки кабинета Супервайзера (СВ)
import { SvMerchandisersTab } from './SvMerchandisersTab';
import { SvStoresTab } from './SvStoresTab';

export default function App() {
  const [authState, setAuthState] = useState<{
    isAuthenticated: boolean;
    phone: string;
    role: string;
    fullName: string;
  }>({
    isAuthenticated: false,
    phone: '',
    role: '',
    fullName: '',
  });

  // Состояния активных вкладок для Владельца (owner)
  const [activeTab, setActiveTab] = useState<'references' | 'questionnaires' | 'info' | 'reports'>('references');
  const [activeRefSubTab, setActiveRefSubTab] = useState<'users' | 'merchandisers' | 'stores' | 'manufacturers' | 'matrices'>('users');

  // Состояния активных вкладок для Супервайзера (supervisor)
  const [activeSvTab, setActiveSvTab] = useState<'merchandisers' | 'stores' | 'reports'>('merchandisers');

  useEffect(() => {
    const savedPhone = sessionStorage.getItem('mg_auth_phone');
    const savedRole = sessionStorage.getItem('mg_auth_role');
    const savedName = sessionStorage.getItem('mg_auth_name');

    if (savedPhone && savedRole && savedName) {
      setAuthState({
        isAuthenticated: true,
        phone: savedPhone,
        role: savedRole,
        fullName: savedName,
      });
    }
  }, []);

  const handleLoginSuccess = (phone: string, password: string, role: string, fullName: string) => {
    sessionStorage.setItem('mg_auth_phone', phone);
    sessionStorage.setItem('mg_auth_password', password);
    sessionStorage.setItem('mg_auth_role', role);
    sessionStorage.setItem('mg_auth_name', fullName);

    setAuthState({
      isAuthenticated: true,
      phone,
      role,
      fullName,
    });
  };

  const handleLogout = () => {
    sessionStorage.clear();
    setAuthState({
      isAuthenticated: false,
      phone: '',
      role: '',
      fullName: '',
    });
  };

  if (!authState.isAuthenticated) {
    return <Login onLoginSuccess={handleLoginSuccess} />;
  }

  // Общие scannable-стили навигации
  const tabStyle = (isActive: boolean) => ({
    padding: '10px 20px',
    cursor: 'pointer',
    backgroundColor: isActive ? '#0275d8' : '#fff',
    color: isActive ? '#fff' : '#333',
    border: '1px solid #ccc',
    borderRadius: '4px',
    marginRight: '10px',
    fontWeight: 'bold' as const,
  });

  const subTabStyle = (isActive: boolean) => ({
    padding: '8px 15px',
    cursor: 'pointer',
    backgroundColor: isActive ? '#5cb85c' : '#f8f9fa',
    color: isActive ? '#fff' : '#333',
    border: '1px solid #ddd',
    borderRadius: '4px',
    marginRight: '8px',
    fontSize: '14px',
  });

    return (
    <div style={{ fontFamily: 'sans-serif', padding: '20px', maxWidth: '1200px', margin: '0 auto' }}>
      {/* Шапка ЛК */}
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '2px solid #0275d8', paddingBottom: '15px', marginBottom: '20px' }}>
        <div>
          <h1 style={{ margin: 0, fontSize: '26px', color: '#333' }}>Панель управления MG SFA</h1>
          <p style={{ margin: '5px 0 0 0', color: '#666' }}>
            Пользователь: <strong>{authState.fullName}</strong> | Роль: <span style={{ color: '#0275d8', fontWeight: 'bold' }}>{authState.role}</span>
          </p>
        </div>
        <button onClick={handleLogout} style={{ padding: '8px 15px', borderRadius: '4px', border: '1px solid #d9534f', backgroundColor: '#fff', color: '#d9534f', cursor: 'pointer', fontWeight: 'bold' }}>
          Выйти
        </button>
      </header>

      {/* ==================================================================== */}
      {/* КОНТУР ИНТЕРФЕЙСА ПРОИЗВОДИТЕЛЯ (ROLE: MANUFACTURER)                 */}
      {/* ==================================================================== */}
      {authState.role === 'manufacturer' ? (
        <div>
          <nav style={{ display: 'flex', marginBottom: '20px' }}>
            <button style={tabStyle(true)}>Отчёты по бренду</button>
          </nav>
          <div style={{ backgroundColor: '#fff', border: '1px solid #ddd', borderRadius: '6px', padding: '20px', minHeight: '400px' }}>
            {/* По ТЗ производитель видит ТОЛЬКО раздел отчетов */}
            <ReportsTab />
          </div>
        </div>
      ) : authState.role === 'supervisor' ? (
        /* ==================================================================== */
        /* КОНТУР ИНТЕРФЕЙСА СУПЕРВАЙЗЕРА (ROLE: SUPERVISOR)                     */
        /* ==================================================================== */
        <div>
          <nav style={{ display: 'flex', marginBottom: '20px' }}>
            <button style={tabStyle(activeSvTab === 'merchandisers')} onClick={() => setActiveSvTab('merchandisers')}>Мерчандайзеры</button>
            <button style={tabStyle(activeSvTab === 'stores')} onClick={() => setActiveSvTab('stores')}>Точки</button>
            <button style={tabStyle(activeSvTab === 'reports')} onClick={() => setActiveSvTab('reports')}>Отчёты</button>
          </nav>

          <div style={{ backgroundColor: '#fff', border: '1px solid #ddd', borderRadius: '6px', padding: '20px', minHeight: '400px' }}>
            {activeSvTab === 'merchandisers' && <SvMerchandisersTab />}
            {activeSvTab === 'stores' && <SvStoresTab />}
            {activeSvTab === 'reports' && <ReportsTab />}
          </div>
        </div>
      ) : (
        /* ==================================================================== */
        /* КОНТУР ИНТЕРФЕЙСА ВЛАДЕЛЬЦА (ROLE: OWNER)                           */
        /* ==================================================================== */
        <div>
          <nav style={{ display: 'flex', marginBottom: '20px' }}>
            <button style={tabStyle(activeTab === 'references')} onClick={() => setActiveTab('references')}>Справочники</button>
            <button style={tabStyle(activeTab === 'questionnaires')} onClick={() => setActiveTab('questionnaires')}>Анкеты</button>
            <button style={tabStyle(activeTab === 'info')} onClick={() => setActiveTab('info')}>Инфо</button>
            <button style={tabStyle(activeTab === 'reports')} onClick={() => setActiveTab('reports')}>Отчеты</button>
          </nav>

          <div style={{ backgroundColor: '#fff', border: '1px solid #ddd', borderRadius: '6px', padding: '20px', minHeight: '400px' }}>
            {activeTab === 'references' && (
              <div>
                <div style={{ display: 'flex', borderBottom: '1px solid #eee', paddingBottom: '12px', marginBottom: '20px' }}>
                  <button style={subTabStyle(activeRefSubTab === 'users')} onClick={() => setActiveRefSubTab('users')}>Пользователи</button>
                  <button style={subTabStyle(activeRefSubTab === 'merchandisers')} onClick={() => setActiveRefSubTab('merchandisers')}>Мерчандайзеры</button>
                  <button style={subTabStyle(activeRefSubTab === 'stores')} onClick={() => setActiveRefSubTab('stores')}>Точки</button>
                  <button style={subTabStyle(activeRefSubTab === 'manufacturers')} onClick={() => setActiveRefSubTab('manufacturers')}>Производители</button>
                  <button style={subTabStyle(activeRefSubTab === 'matrices')} onClick={() => setActiveRefSubTab('matrices')}>Матрицы</button>
                </div>
                {activeRefSubTab === 'users' && <UsersTab />}
                {activeRefSubTab === 'merchandisers' && <MerchandisersTab />}
                {activeRefSubTab === 'stores' && <StoresTab />}
                {activeRefSubTab === 'manufacturers' && <ManufacturersTab />}
                {activeRefSubTab === 'matrices' && <MatricesTab />}
              </div>
            )}
            {activeTab === 'questionnaires' && <QuestionnairesTab />}
            {activeTab === 'info' && <InfoTab />}
            {activeTab === 'reports' && <ReportsTab />}
          </div>
        </div>
      )}
    </div>
  );
}
