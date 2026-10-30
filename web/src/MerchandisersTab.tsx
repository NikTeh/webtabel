import React, { useState } from 'react';

export const MerchandisersTab: React.FC = () => {
  const [file, setFile] = useState<File | null>(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setMessage('');
      setError('');
    }
  };

  const handleImport = async () => {
    if (!file) {
      setError('Пожалуйста, выберите файл для импорта');
      return;
    }

    setUploading(true);
    setMessage('');
    setError('');

    const formData = new FormData();
    formData.append('file', file);

    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);

    try {
      const response = await fetch('/api/v1/owner/merchandisers/import', {
        method: 'POST',
        headers: {
          'Authorization': `Basic ${credentials}`,
        },
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        setError(data.error || 'Ошибка при импорте файла на сервере');
        return;
      }

      setMessage(`Успешно импортировано мерчандайзеров: ${data.imported_merchandisers_count}`);
      setFile(null);
    } catch (err) {
      setError('Не удалось отправить файл на сервер бэкенда');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Справочник мерчандайзеров агентства</h3>
      
      {/* Контур импорта XLSX таблицы мерчандайзеров (Пункт 8 ТЗ) */}
      <div style={{ backgroundColor: '#f8f9fa', border: '1px dashed #ccc', padding: '20px', borderRadius: '6px', marginBottom: '20px', maxWidth: '500px' }}>
        <h4 style={{ margin: '0 0 15px 0' }}>Импорт XLSX таблицы мерчандайзеров</h4>
        
        {message && <div style={{ color: '#5cb85c', marginBottom: '10px', fontWeight: 'bold' }}>{message}</div>}
        {error && <div style={{ color: '#d9534f', marginBottom: '10px', fontWeight: 'bold' }}>{error}</div>}

        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <input type="file" accept=".xlsx" onChange={handleFileChange} style={{ fontSize: '14px' }} />
          <button onClick={handleImport} disabled={uploading || !file} style={{ padding: '8px 15px', backgroundColor: '#5cb85c', color: '#fff', border: 'none', borderRadius: '4px', cursor: (uploading || !file) ? 'not-allowed' : 'pointer', fontWeight: 'bold' }}>
            {uploading ? 'Загрузка...' : 'Импортировать'}
          </button>
        </div>
        <small style={{ color: '#777', display: 'block', marginTop: '10px' }}>Максимальный размер: 10 МБ. Лимит: 2000 строк.</small>
      </div>

      {/* Заглушка для таблицы по ТЗ */}
      <div style={{ marginTop: '20px', border: '1px solid #eee', padding: '15px', borderRadius: '4px' }}>
        <p style={{ color: '#777', fontStyle: 'italic', margin: 0 }}>Таблица просмотра, ручного добавления, изменения и удаления строк мерчандайзеров.</p>
      </div>
    </div>
  );
};
