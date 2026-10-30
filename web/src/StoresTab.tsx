import React, { useState, useEffect } from 'react';

interface StoreRow {
  id_1c: string;
  region: string;
  city: string;
  retail_network: string;
  store_sign: string;
  store_format: string;
  address: string;
  sv_name: string;
  sv_phone: string;
}

export const StoresTab: React.FC = () => {
  const [stores, setStores] = useState<StoreRow[]>([]);
  const [file, setFile] = useState<File | null>(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);
  const [loading, setLoading] = useState(false);

  // Получение учетных данных авторизации из сессии браузера
  const getHeaders = () => {
    const phone = sessionStorage.getItem('mg_auth_phone') || '';
    const password = sessionStorage.getItem('mg_auth_password') || '';
    const credentials = btoa(`${phone}:${password}`);
    return {
      'Authorization': `Basic ${credentials}`
    };
  };

  // Функция для скачивания реального глобального списка точек из базы данных бэкенда
  const loadStoresFromDb = async () => {
    setLoading(true);
    try {
      // Отправляем GET-запрос на системный маршрут сбора данных аналитики
      const response = await fetch('/api/v1/owner/stores/import', {
        method: 'GET',
        headers: getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        setStores(data || []);
      }
    } catch {
      setError('Не удалось подключиться к серверу для чтения списка магазинов');
    } finally {
      setLoading(false);
    }
  };

  // Автоматически загружаем список магазинов при открытии вкладки
  useEffect(() => {
    loadStoresFromDb();
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setMessage('');
      setError('');
    }
  };

  // Обработчик отправки XLSX файла на сервер по принципу Умного импорта
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

    try {
      const response = await fetch('/api/v1/owner/stores/import', {
        method: 'POST',
        headers: {
          'Authorization': getHeaders()['Authorization']
        },
        body: formData
      });

      const data = await response.json();

      if (!response.ok) {
        setError(data.error || 'Ошибка при обработке файла Excel на сервере');
        return;
      }

      // Выводим детальный промышленный статус обработки строк
      setMessage(`Импорт успешно завершен! Добавлено новых точек: ${data.inserted}, автоматически обновлено существующих: ${data.updated}`);
      setFile(null);
      
      // Перезапускаем чтение из базы, чтобы обновить таблицу на экране владельца
      loadStoresFromDb();
    } catch (err) {
      setError('Ошибка отправки файла на сервер бэкенда');
    } finally {
      setUploading(false);
    }
  };
  return (
    <div style={{ padding: '10px 0' }}>
      <h3>Глобальный справочник торговых точек агентства</h3>
      
      {/* Панель Умного импорта XLSX файлов */}
      <div style={{ backgroundColor: '#f8f9fa', border: '1px dashed #ccc', padding: '20px', borderRadius: '6px', marginBottom: '25px', maxWidth: '650px' }}>
        <h4 style={{ margin: '0 0 15px 0' }}>Массовое обновление адресной программы (Excel)</h4>
        
        {message && <div style={{ color: '#137333', backgroundColor: '#e6F4ea', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold', fontSize: '14px' }}>{message}</div>}
        {error && <div style={{ color: '#c5221f', backgroundColor: '#fce8e6', padding: '10px', borderRadius: '4px', marginBottom: '15px', fontWeight: 'bold', fontSize: '14px' }}>{error}</div>}

      
       <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <input type="file" accept=".xlsx" onChange={handleFileChange} style={{ fontSize: '14px' }} />
          <button onClick={handleImport} disabled={uploading || !file} style={{ padding: '8px 15px', backgroundColor: '#f0ad4e', color: '#fff', border: 'none', borderRadius: '4px', cursor: (uploading || !file) ? 'not-allowed' : 'pointer', fontWeight: 'bold' }}>
            {uploading ? 'Загрузка...' : 'Импортировать'}
          </button>
        </div>
        <small style={{ color: '#777', display: 'block', marginTop: '10px' }}>Максимальный размер: 20 МБ. Лимит: 2000 строк. Формат колонок: ID ТТ, Регион, Город, Сеть, Вывеска, Формат, Адрес, ФИО СВ, Телефон СВ, Широта, Долгота.</small>
      </div>

      {/* Таблица живого вывода данных из СУБД PostgreSQL */}
      <div style={{ marginTop: '20px' }}>
        <h4 style={{ marginBottom: '15px', color: '#333' }}>Текущий реестр магазинов в базе данных ({stores.length})</h4>
        
        {loading ? (
          <p style={{ color: '#666', fontStyle: 'italic' }}>Загрузка строк из базы данных PostgreSQL...</p>
        ) : stores.length === 0 ? (
          <div style={{ color: '#777', padding: '20px', backgroundColor: '#f9f9f9', borderRadius: '4px', border: '1px solid #eee' }}>
            В базе данных агентства еще нет ни одного магазина. Загрузите первый файл Excel выше.
          </div>
        ) : (
          <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: '6px' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
              <thead>
                <tr style={{ backgroundColor: '#f8f9fa', borderBottom: '2px solid #ddd' }}>
                  <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Код 1С (PK)</th>
                  <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Регион / Город</th>
                  <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Торговая сеть</th>
                  <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Магазин / Формат</th>
                  <th style={{ padding: '10px', borderRight: '1px solid #ddd' }}>Фактический адрес</th>
                  <th style={{ padding: '10px' }}>Закрепленный Супервайзер</th>
                </tr>
              </thead>
              <tbody>
                {stores.map((row) => (
                  <tr key={row.id_1c} style={{ borderBottom: '1px solid #eee' }}>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}><code>{row.id_1c}</code></td>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>{row.region}, <strong>{row.city}</strong></td>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd', color: '#0275d8', fontWeight: 'bold' }}>{row.retail_network}</td>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>{row.store_sign} <br/><span style={{ fontSize: '11px', color: '#777' }}>{row.store_format}</span></td>
                    <td style={{ padding: '10px', borderRight: '1px solid #ddd' }}>{row.address}</td>
                    <td style={{ padding: '10px', color: '#555' }}>{row.sv_name} <br/><span style={{ fontSize: '11px', color: '#888' }}>{row.sv_phone}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
