import React, { useState } from 'react';

interface LoginProps {
  onLoginSuccess: (phone: string, password: string, role: string, fullName: string) => void;
}

export const Login: React.FC<LoginProps> = ({ onLoginSuccess }) => {
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    // ВРЕМЕННЫЙ ДЕМО-ВХОД ДЛЯ ТЕСТИРОВАНИЯ ИНТЕРФЕЙСА БЕЗ БД
    if (phone.trim() === 'admin' && password === 'admin') {
      onLoginSuccess('admin', 'admin', 'owner', 'Владелец (Тест)');
      return;
    }

        // ДОБАВИТЬ ЭТОТ БЛОК ДЛЯ ТЕСТА КАБИНЕТА СВ
    if (phone.trim() === 'sv' && password === 'sv') {
      onLoginSuccess('sv', 'sv', 'supervisor', 'Иванов И.И. (Супервайзер)');
      return;
    }

        // ДОБАВИТЬ ЭТОТ БЛОК ДЛЯ ТЕСТА КАБИНЕТА ПРОИЗВОДИТЕЛЯ
    if (phone.trim() === 'client' && password === 'client') {
      onLoginSuccess('client', 'client', 'manufacturer', 'Представитель Coca-Cola');
      return;
    }

    setLoading(true);
    const credentials = btoa(`${phone.trim()}:${password}`);

    try {
      const response = await fetch('/api/v1/test/protected', {
        method: 'GET',
        headers: {
          'Authorization': `Basic ${credentials}`,
          'Content-Type': 'application/json',
        },
      });

      if (response.status === 401) {
        setError('Неверный номер телефона или пароль');
        setLoading(false);
        return;
      }

      if (!response.ok) {
        setError('Ошибка сервера при авторизации');
        setLoading(false);
        return;
      }

      const data = await response.json();
      
      if (data.role === 'merchandiser') {
        setError('Доступ к веб-версии для мерчандайзеров запрещен. Используйте Android приложение.');
        setLoading(false);
        return;
      }

      onLoginSuccess(phone.trim(), password, data.role, data.name || 'Пользователь');
    } catch (err) {
      setError('Не удалось связаться с сервером.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', backgroundColor: '#f5f5f5', fontFamily: 'sans-serif' }}>
      <form onSubmit={handleSubmit} style={{ padding: '30px', borderRadius: '8px', backgroundColor: '#ffffff', boxShadow: '0 4px 6px rgba(0,0,0,0.1)', width: '320px' }}>
        <h2 style={{ textAlign: 'center', marginBottom: '24px', color: '#333' }}>Вход в систему MG</h2>
        
        {error && <div style={{ color: '#d9534f', backgroundColor: '#fdf7f7', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontSize: '14px', textAlign: 'center' }}>{error}</div>}
        
        <div style={{ marginBottom: '15px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', color: '#666' }}>Телефон (Логин):</label>
          <input type="text" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="Для теста введите admin" required style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #ccc', boxSizing: 'border-box' }} />
        </div>

        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '14px', color: '#666' }}>Пароль:</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Для теста введите admin" required style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #ccc', boxSizing: 'border-box' }} />
        </div>

        <button type="submit" disabled={loading} style={{ width: '100%', padding: '10px', borderRadius: '4px', border: 'none', backgroundColor: '#0275d8', color: '#fff', fontSize: '16px', cursor: loading ? 'not-allowed' : 'pointer' }}>
          {loading ? 'Проверка...' : 'Войти'}
        </button>
      </form>
    </div>
  );
};
