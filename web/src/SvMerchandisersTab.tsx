import React, { useState, useEffect } from 'react';

interface Merchandiser {
  id: number;
  external_id: number;
  full_name: string;
  phone: string;
  password: string;
}

export const SvMerchandisersTab: React.FC = () => {
  const [mes, setMes] = useState<Merchandiser[]>([]);
  
  // Состояния для формы создания нового МЕ
  const [fullName, setFullName] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [externalId, setExternalId] = useState(''); 
  
  // Состояния для inline-редактирования строки МЕ
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editFullName, setEditFullName] = useState('');
  const [editPhone, setEditPhone] = useState('');
  const [editPassword, setEditPassword] = useState('');
  const [editExternalId, setEditExternalId] = useState('');

  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const getHeaders = () => {
    const savedPhone = sessionStorage.getItem('mg_auth_phone') || '';
    const savedPassword = sessionStorage.getItem('mg_auth_password') || '';
    return {
      'Authorization': `Basic ${btoa(`${savedPhone}:${savedPassword}`)}`,
      'Content-Type': 'application/json'
    };
  };

  // Чтение списка только своих мерчандайзеров из PostgreSQL
  const loadMes = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/sv/merchandisers', { 
        method: 'GET', 
        headers: { 'Authorization': getHeaders()['Authorization'] } 
      });
      if (response.ok) {
        const data = await response.json();
        setMes(data || []);
      }
    } catch {
      setError('Ошибка получения списка вашей команды с сервера');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMes();
  }, []);
  // Ручное построчное добавление МЕ в свою команду
  const handleAddMe = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage(''); setError('');

    const parsedExtId = parseInt(externalId.trim(), 10);

    if (!fullName.trim() || !phone.trim() || !password.trim() || !externalId.trim() || isNaN(parsedExtId)) {
      setError('Перед сохранением все обязательные ячейки должны быть заполнены (включая цифровой ID из 1С)');
      return;
    }

    try {
      const response = await fetch('/api/v1/sv/merchandisers', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ 
          full_name: fullName.trim(), 
          phone: phone.trim(), 
          password: password.trim(),
          external_id: parsedExtId
        })
      });
      const data = await response.json();
      if (!response.ok) { setError(data.error || 'Ошибка создания МЕ'); return; }

      setMessage('Мерчандайзер успешно добавлен в вашу команду!');
      setFullName(''); setPhone(''); setPassword(''); setExternalId('');
      loadMes();
    } catch { 
      setError('Ошибка соединения с бэкендом при добавлении');
    }
  };

  const startEdit = (row: Merchandiser) => {
    setEditingId(row.id);
    setEditFullName(row.full_name);
    setEditPhone(row.phone);
    setEditPassword(row.password);
    setEditExternalId(row.external_id !== undefined && row.external_id !== null ? String(row.external_id) : '');
  };

  // Построчное сохранение изменений вживую в СУБД
  const handleSaveEdit = async (id: number) => {
    setMessage(''); setError('');

    const currentExtIdStr = editExternalId !== undefined && editExternalId !== null ? String(editExternalId) : '';
    const parsedExtId = parseInt(currentExtIdStr.trim(), 10);

    if (!editFullName.trim() || !editPhone.trim() || !editPassword.trim() || !currentExtIdStr.trim() || isNaN(parsedExtId)) {
      setError('Перед сохранением все обязательные ячейки должны быть заполнены');
      return;
    }

    try {
      const response = await fetch('/api/v1/sv/merchandisers', {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ 
          id, 
          full_name: editFullName.trim(), 
          phone: editPhone.trim(), 
          password: editPassword.trim(),
          external_id: parsedExtId
        })
      });

      if (!response.ok) {
        const data = await response.json();
        setError(data.error || 'Ошибка при изменении строки МЕ');
        return;
      }

      setMessage('Данные мерчандайзера успешно изменены в базе данных');
      setEditingId(null);
      loadMes();
    } catch {
      setError('Ошибка при отправке изменений на сервер');
    }
  };

  const handleDeleteMe = async (id: number) => {
    if (!window.confirm('Вы уверены, что хотите удалить мерчандайзера из команды?')) return;
    setMessage(''); setError('');
    try {
      const response = await fetch(`/api/v1/sv/merchandisers?id=${id}`, { 
        method: 'DELETE', 
        headers: { 'Authorization': getHeaders()['Authorization'] } 
      });
      if (!response.ok) { 
        const data = await response.json(); 
        setError(data.error || 'Ошибка удаления'); 
        return; 
      }
      setMessage('Мерчандайзер успешно удален из вашей команды.');
      loadMes();
    } catch { 
      setError('Ошибка удаления сотрудника');
    }
  };
  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Управление командой мерчандайзеров (Полный CRUD)</h3>
      
      {message && <div style={{ color: '#137333', backgroundColor: '#e6f4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{message}</div>}
      {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{error}</div>}

      <form onSubmit={handleAddMe} style={{ display: 'flex', gap: '10px', marginBottom: '25px', flexWrap: 'wrap', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #ddd' }}>
        <input type="text" placeholder="Код 1С (Число)" value={externalId} onChange={e => setExternalId(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px', width: '130px' }} />
        <input type="text" placeholder="ФИО Мерчандайзера" value={fullName} onChange={e => setFullName(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px', width: '220px' }} />
        <input type="text" placeholder="Телефон (Логин)" value={phone} onChange={e => setPhone(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px', width: '160px' }} />
        <input type="text" placeholder="Пароль" value={password} onChange={e => setPassword(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px', width: '140px' }} />
        <button type="submit" style={{ padding: '8px 20px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>+ Добавить МЕ</button>
      </form>

      {loading ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>Загрузка списка сотрудников из PostgreSQL...</p>
      ) : mes.length === 0 ? (
        <div style={{ color: '#777', padding: '20px', backgroundColor: '#f9f9f9', borderRadius: '4px', border: '1px solid #eee' }}>
          В вашей команде еще нет мерчандайзеров. Добавьте первого сотрудника через форму выше.
        </div>
      ) : (
        <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: '6px' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px', textAlign: 'left' }}>
            <thead>
              <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '120px' }}>ID из 1С</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd' }}>ФИО сотрудника</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '150px' }}>Телефон</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '160px' }}>Пароль (Открытый)</th>
                <th style={{ padding: '12px', textAlign: 'center', width: '200px' }}>Действия</th>
              </tr>
            </thead>
            <tbody>
              {mes.map(row => {
                const isEditing = editingId === row.id;
                return (
                  <tr key={row.id} style={{ borderBottom: '1px solid #eee', backgroundColor: isEditing ? '#fbf8ef' : '#fff' }}>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editExternalId} onChange={e => setEditExternalId(e.target.value)} style={{ width: '90%', padding: '5px', border: '1px solid #0275d8', borderRadius: '4px' }} />
                      ) : (
                        <code>{row.external_id || '—'}</code>
                      )}
                    </td>
                    
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editFullName} onChange={e => setEditFullName(e.target.value)} style={{ width: '95%', padding: '5px', border: '1px solid #0275d8', borderRadius: '4px' }} />
                      ) : (
                        <strong style={{ color: '#333' }}>{row.full_name}</strong>
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editPhone} onChange={e => setEditPhone(e.target.value)} style={{ width: '95%', padding: '5px', border: '1px solid #0275d8', borderRadius: '4px' }} />
                      ) : (
                        row.phone
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editPassword} onChange={e => setEditPassword(e.target.value)} style={{ width: '95%', padding: '5px', border: '1px solid #0275d8', borderRadius: '4px' }} />
                      ) : (
                        <code>{row.password}</code>
                      )}
                    </td>

                    <td style={{ padding: '10px', textAlign: 'center' }}>
                      {isEditing ? (
                        <div style={{ display: 'flex', gap: '5px', justifyContent: 'center' }}>
                          <button onClick={() => handleSaveEdit(row.id)} style={{ padding: '4px 10px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontSize: '12px', fontWeight: 'bold' }}>Сохранить</button>
                          <button onClick={() => setEditingId(null)} style={{ padding: '4px 10px', backgroundColor: '#fff', color: '#777', border: '1px solid #ccc', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}>Отмена</button>
                        </div>
                      ) : (
                        <div style={{ display: 'flex', gap: '5px', justifyContent: 'center' }}>
                          <button onClick={() => startEdit(row)} style={{ padding: '4px 10px', backgroundColor: '#fff', color: '#0275d8', border: '1px solid #0275d8', borderRadius: '4px', cursor: 'pointer', fontSize: '12px', fontWeight: 'bold' }}>Редактировать</button>
                          <button onClick={() => handleDeleteMe(row.id)} style={{ padding: '4px 10px', backgroundColor: '#fff', color: '#d9534f', border: '1px solid #d9534f', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}>Удалить</button>
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
