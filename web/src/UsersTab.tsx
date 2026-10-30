import React, { useState, useEffect } from 'react';

interface User {
  id: number;
  external_id: number | null;
  phone: string;
  password: string;
  full_name: string;
  role: string;
  manufacturer_id: number | null;
}

interface ManufacturerOption {
  id: number;
  name: string;
}

export const UsersTab: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [manufacturers, setManufacturers] = useState<ManufacturerOption[]>([]);
  const [file, setFile] = useState<File | null>(null);
  
  // Состояния для формы ручного создания одного пользователя
  const [fullName, setFullName] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState('supervisor');
  const [externalId, setExternalId] = useState('');
  const [manufacturerId, setManufacturerId] = useState('');

  // Состояния для построчного inline-редактирования
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editFullName, setEditFullName] = useState('');
  const [editPhone, setEditPhone] = useState('');
  const [editPassword, setEditPassword] = useState('');
  const [editRole, setEditRole] = useState('');
  const [editExternalId, setEditExternalId] = useState('');
  const [editManufacturerId, setEditManufacturerId] = useState('');

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

  // Загрузка реального списка всех производителей для выпадающих списков связывания
  const loadManufacturers = async () => {
    try {
      const response = await fetch('/api/v1/owner/manufacturers', {
        method: 'GET',
        headers: { 'Authorization': getHeaders()['Authorization'] }
      });
      if (response.ok) {
        const data = await response.json();
        // Поддерживаем формат массива карточек [{id: 1, name: "Coca-Cola"}]
        setManufacturers(data || []);
      }
    } catch {
      console.error('Ошибка загрузки справочника клиентов');
    }
  };

  // Загрузка списка всех пользователей из базы данных бэкенда
  const loadUsers = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/owner/users', { 
        method: 'GET', 
        headers: { 'Authorization': getHeaders()['Authorization'] }
      });
      if (response.ok) {
        const data = await response.json();
        setUsers(data || []);
      }
    } catch {
      setError('Ошибка подключения к серверу при чтении пользователей');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
    loadManufacturers();
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setMessage('');
      setError('');
    }
  };

  // Массовая загрузка пользователей через Умный импорт
  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage(''); setError('');

    if (!file) {
      setError('Выберите файл Excel для импорта');
      return;
    }

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await fetch('/api/v1/owner/users/import', {
        method: 'POST',
        headers: { 'Authorization': getHeaders()['Authorization'] },
        body: formData
      });
      const data = await response.json();

      if (!response.ok) {
        setError(data.error || 'Ошибка при импорте пользователей');
        return;
      }

      setMessage(`Добавлено новых сотрудников: ${data.inserted}, обновлено существующих: ${data.updated}`);
      setFile(null);
      const fileInput = document.getElementById('user-file-input') as HTMLInputElement;
      if (fileInput) fileInput.value = '';
      
      loadUsers();
    } catch {
      setError('Ошибка отправки файла на сервер');
    }
  };
  // Ручное создание одной учетной записи сотрудника руками Владельца
  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage(''); setError('');

    const parsedExtId = externalId.trim() ? parseInt(externalId.trim(), 10) : null;
    const parsedManId = role === 'manufacturer' && manufacturerId ? parseInt(manufacturerId, 10) : null;

    if (!fullName.trim() || !phone.trim() || !password.trim() || !role) {
      setError('Заполните обязательные поля: ФИО, Телефон, Пароль и Роль');
      return;
    }

    try {
      const response = await fetch('/api/v1/owner/users', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          full_name: fullName.trim(),
          phone: phone.trim(),
          password: password.trim(),
          role: role,
          external_id: parsedExtId,
          manufacturer_id: parsedManId
        })
      });
      const data = await response.json();
      if (!response.ok) { setError(data.error || 'Ошибка создания пользователя'); return; }

      setMessage('Пользователь успешно создан в базе данных!');
      setFullName(''); setPhone(''); setPassword(''); setExternalId(''); setManufacturerId('');
      loadUsers();
    } catch {
      setError('Ошибка ручного добавления пользователя');
    }
  };

  const startEdit = (row: User) => {
    setEditingId(row.id);
    setEditFullName(row.full_name);
    setEditPhone(row.phone);
    setEditPassword(row.password);
    setEditRole(row.role);
    setEditExternalId(row.external_id !== null && row.external_id !== undefined ? String(row.external_id) : '');
    setEditManufacturerId(row.manufacturer_id !== null && row.manufacturer_id !== undefined ? String(row.manufacturer_id) : '');
  };

  const cancelEdit = () => {
    setEditingId(null);
  };
  // Построчное сохранение изменений вживую на экране в PostgreSQL
  const handleSaveEdit = async (id: number) => {
    setMessage(''); setError('');

    const currentExtIdStr = editExternalId !== undefined && editExternalId !== null ? String(editExternalId) : '';
    const parsedExtId = currentExtIdStr.trim() ? parseInt(currentExtIdStr.trim(), 10) : null;

    const currentManIdStr = editManufacturerId !== undefined && editManufacturerId !== null ? String(editManufacturerId) : '';
    const parsedManId = editRole === 'manufacturer' && currentManIdStr.trim() ? parseInt(currentManIdStr.trim(), 10) : null;

    if (!editFullName.trim() || !editPhone.trim() || !editPassword.trim() || !editRole) {
      setError('Заполните обязательные поля: ФИО, Телефон, Пароль и Роль');
      return;
    }

    try {
      const response = await fetch('/api/v1/owner/users', {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({
          id,
          full_name: editFullName.trim(),
          phone: editPhone.trim(),
          password: editPassword.trim(),
          role: editRole,
          external_id: parsedExtId,
          manufacturer_id: parsedManId
        })
      });

      if (!response.ok) {
        const data = await response.json();
        setError(data.error || 'Ошибка при изменении данных');
        return;
      }

      setMessage('Данные пользователя успешно изменены');
      setEditingId(null);
      loadUsers();
    } catch {
      setError('Ошибка сохранения изменений');
    }
  };

  const handleDeleteUser = async (id: number) => {
    if (!window.confirm('Вы уверены, что хотите полностью удалить пользователя из системы?')) return;
    setMessage(''); setError('');
    try {
      const response = await fetch(`/api/v1/owner/users?id=${id}`, { 
        method: 'DELETE', 
        headers: { 'Authorization': getHeaders()['Authorization'] } 
      });
      if (!response.ok) { 
        const data = await response.json(); 
        setError(data.error || 'Ошибка удаления'); 
        return; 
      }
      setMessage('Пользователь успешно удален из системы.');
      loadUsers();
    } catch {
      setError('Ошибка при отправке запроса на удаление');
    }
  };

  const translateRole = (role: string) => {
    switch (role) {
      case 'owner': return 'Владелец';
      case 'supervisor': return 'Супервайзер';
      case 'merchandiser': return 'Мерчандайзер';
      case 'manufacturer': return 'Производитель';
      default: return role;
    }
  };
  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Справочник пользователей системы</h3>
      
      {message && <div style={{ color: '#137333', backgroundColor: '#e6f4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{message}</div>}
      {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold' }}>{error}</div>}

      <div style={{ display: 'flex', gap: '20px', marginBottom: '25px', flexWrap: 'wrap' }}>
        {/* Панель Умного импорта XLSX */}
        <form onSubmit={handleImport} style={{ display: 'flex', flexDirection: 'column', gap: '10px', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #ddd', width: '420px' }}>
          <h4 style={{ margin: '0 0 5px 0' }}>Массовый импорт / Обновление сотрудников</h4>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
            <input id="user-file-input" type="file" accept=".xlsx" onChange={handleFileChange} style={{ padding: '4px' }} />
            <button type="submit" style={{ padding: '8px 15px', backgroundColor: '#0275d8', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>Импортировать</button>
          </div>
          <small style={{ color: '#666' }}>Новые логины добавятся, старые — обновят пароли и ФИО без сноса связей.</small>
        </form>

        {/* Форма построчного добавления одной строки руками Владельца */}
        <form onSubmit={handleAddUser} style={{ display: 'flex', gap: '10px', backgroundColor: '#f8f9fa', padding: '15px', borderRadius: '6px', border: '1px solid #ddd', flexWrap: 'wrap', flex: '1' }}>
          <h4 style={{ margin: '0 0 5px 0', width: '100%' }}>Ручное точечное добавление</h4>
          <input type="text" placeholder="Код 1С" value={externalId} onChange={e => setExternalId(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '80px' }} />
          <input type="text" placeholder="ФИО сотрудника" value={fullName} onChange={e => setFullName(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '160px' }} />
          <input type="text" placeholder="Телефон" value={phone} onChange={e => setPhone(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '130px' }} />
          <input type="text" placeholder="Пароль" value={password} onChange={e => setPassword(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '100px' }} />
          
          <select value={role} onChange={e => setRole(e.target.value)} style={{ padding: '6px', border: '1px solid #ccc', borderRadius: '4px', width: '130px' }}>
            <option value="supervisor">Супервайзер</option>
            <option value="merchandiser">Мерчандайзер</option>
            <option value="manufacturer">Производитель</option>
          </select>

          {role === 'manufacturer' && (
            <select value={manufacturerId} onChange={e => setManufacturerId(e.target.value)} style={{ padding: '6px', border: '1px solid #0275d8', borderRadius: '4px', width: '160px', fontWeight: 'bold' }}>
              <option value="">-- Выберите бренд --</option>
              {manufacturers.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
            </select>
          )}

          <button type="submit" style={{ padding: '6px 15px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>+ Добавить</button>
        </form>
      </div>

      {/* Интерактивная CRUD таблица */}
      {loading ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>Чтение данных пользователей из PostgreSQL...</p>
      ) : users.length === 0 ? (
        <div style={{ color: '#777', padding: '20px', backgroundColor: '#f9f9f9', borderRadius: '4px', border: '1px solid #eee' }}>
          Пользователи в базе данных не найдены. Проведите первый импорт.
        </div>
      ) : (
        <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: '6px' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px', textAlign: 'left' }}>
            <thead>
              <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '100px' }}>Код 1С</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd' }}>ФИО пользователя</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '140px' }}>Телефон (Логин)</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '140px' }}>Пароль</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '150px' }}>Системная Роль</th>
                <th style={{ padding: '12px', borderRight: '1px solid #ddd', width: '160px' }}>Привязанный клиент</th>
                <th style={{ padding: '12px', textAlign: 'center', width: '180px' }}>Действия</th>
              </tr>
            </thead>
            <tbody>
              {users.map(row => {
                const isEditing = editingId === row.id;
                return (
                  <tr key={row.id} style={{ borderBottom: '1px solid #eee', backgroundColor: '#fff' }}>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editExternalId} onChange={e => setEditExternalId(e.target.value)} style={{ width: '85%', padding: '4px' }} />
                      ) : (
                        <code>{row.external_id || '—'}</code>
                      )}
                    </td>
                    
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editFullName} onChange={e => setEditFullName(e.target.value)} style={{ width: '90%', padding: '4px' }} />
                      ) : (
                        <strong style={{ color: '#333' }}>{row.full_name}</strong>
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editPhone} onChange={e => setEditPhone(e.target.value)} style={{ width: '90%', padding: '4px' }} />
                      ) : (
                        row.phone
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <input type="text" value={editPassword} onChange={e => setEditPassword(e.target.value)} style={{ width: '90%', padding: '4px' }} />
                      ) : (
                        <code>{row.password}</code>
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing ? (
                        <select value={editRole} onChange={e => setEditRole(e.target.value)} style={{ padding: '4px', width: '100%' }}>
                          <option value="supervisor">Супервайзер</option>
                          <option value="merchandiser">Мерчандайзер</option>
                          <option value="manufacturer">Производитель</option>
                        </select>
                      ) : (
                        <span style={{ 
                          padding: '3px 8px', borderRadius: '12px', fontSize: '11px', fontWeight: 'bold',
                          backgroundColor: row.role === 'supervisor' ? '#d1ecf1' : row.role === 'merchandiser' ? '#fff3cd' : '#e2e3e5',
                          color: row.role === 'supervisor' ? '#0c5460' : row.role === 'merchandiser' ? '#856404' : '#383d41'
                        }}>
                          {translateRole(row.role)}
                        </span>
                      )}
                    </td>

                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>
                      {isEditing && editRole === 'manufacturer' ? (
                        <select value={editManufacturerId} onChange={e => setEditManufacturerId(e.target.value)} style={{ padding: '4px', width: '100%' }}>
                          <option value="">-- Выберите --</option>
                          {manufacturers.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
                        </select>
                      ) : (
                        <span style={{ fontSize: '13px', fontWeight: '500', color: row.manufacturer_id ? '#0275d8' : '#666' }}>
                          {row.manufacturer_id ? manufacturers.find(m => m.id === row.manufacturer_id)?.name || `ID: ${row.manufacturer_id}` : '—'}
                        </span>
                      )}
                    </td>

                    <td style={{ padding: '10px', textAlign: 'center' }}>
                      {isEditing ? (
                        <div style={{ display: 'flex', gap: '5px', justifyContent: 'center' }}>
                          <button onClick={() => handleSaveEdit(row.id)} style={{ padding: '4px 8px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer', fontSize: '12px', fontWeight: 'bold' }}>Сохранить</button>
                          <button onClick={cancelEdit} style={{ padding: '4px 8px', backgroundColor: '#fff', color: '#777', border: '1px solid #ccc', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}>Отмена</button>
                        </div>
                      ) : (
                        <div style={{ display: 'flex', gap: '5px', justifyContent: 'center' }}>
                          <button onClick={() => startEdit(row)} style={{ padding: '4px 8px', backgroundColor: '#fff', color: '#0275d8', border: '1px solid #0275d8', borderRadius: '4px', cursor: 'pointer', fontSize: '12px', fontWeight: 'bold' }}>Изменить</button>
                          <button onClick={() => handleDeleteUser(row.id)} style={{ padding: '4px 8px', backgroundColor: '#fff', color: '#d9534f', border: '1px solid #d9534f', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}>Удалить</button>
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
