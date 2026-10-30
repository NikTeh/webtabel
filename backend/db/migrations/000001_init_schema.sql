-- 1. ЕДИНАЯ ТАБЛИЦА ПОЛЬЗОВАТЕЛЕЙ (БЕЗ ШИФРОВАНИЯ ПАРОЛЕЙ ПО ТЗ)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    external_id INT UNIQUE,           -- ID из XLSX файлов Владельца (для СВ и МЕ)
    phone VARCHAR(20) UNIQUE NOT NULL, -- Логин для ЛК браузера / Android приложения
    password TEXT NOT NULL,            -- Пароль в открытом текстовом виде для оперативной поддержки МЕ
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'supervisor', 'merchandiser', 'manufacturer')),
    manufacturer_id INT NULL,          -- Заполняется только для роли 'manufacturer'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. СВЯЗУЮЩАЯ ТАБЛИЦА СУПЕРВАЙЗЕР -> МЕРЧАНДАЙЗЕР (Один СВ имеет много МЕ)
CREATE TABLE supervisor_merchandisers (
    supervisor_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchandiser_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (supervisor_id, merchandiser_id)
);

-- 3. ТАБЛИЦА ПРОИЗВОДИТЕЛЕЙ
CREATE TABLE manufacturers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    work_days INT[] NOT NULL,               -- Массив дней недели {1,3,5} для графиков
    duration_minutes INT NOT NULL DEFAULT 0, -- Суммарный тайминг обслуживания в минутах
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем внешний ключ для связи пользователя-производителя с его карточкой
ALTER TABLE users 
ADD CONSTRAINT fk_users_manufacturer 
FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id) ON DELETE SET NULL;

-- -- 4. ЕДИНЫЙ СПРАВОЧНИК ТОРГОВЫХ ТОЧК АГЕНТСТВА (id_1c — ГЛАВНЫЙ СИСТЕМНЫЙ PK)
-- ...
CREATE TABLE stores (
    id_1c VARCHAR(50) PRIMARY KEY,     -- Уникальный текстовый ID точки из 1С агентства
    region VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    retail_network VARCHAR(100) NOT NULL,
    store_sign VARCHAR(100) NOT NULL,
    store_format VARCHAR(50) NOT NULL,
    address TEXT NOT NULL,
    supervisor_id INT NULL REFERENCES users(id) ON DELETE SET NULL, -- Закрепленный СВ
    merchandiser_id INT NULL REFERENCES users(id) ON DELETE SET NULL, -- Оперативное управление МЕ внутри точки
    latitude NUMERIC(9,6) NULL,                                      -- ДОБАВЛЕНО ДЛЯ ГЕО-БАЗЫ
    longitude NUMERIC(9,6) NULL,                                     -- ДОБАВЛЕНО ДЛЯ ГЕО-БАЗЫ
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- ...


-- 5. ТАБЛИЦА СВЯЗИ ПРОИЗВОДИТЕЛЕЙ С АГЕНТСТВСКИМИ ТОЧКАМИ
CREATE TABLE manufacturer_stores (
    manufacturer_id INT REFERENCES manufacturers(id) ON DELETE CASCADE,
    store_id_1c VARCHAR(50) REFERENCES stores(id_1c) ON DELETE CASCADE,
    PRIMARY KEY (manufacturer_id, store_id_1c)
);

-- 6. ТАБЛИЦА SKU ПРОИЗВОДИТЕЛЕЙ
CREATE TABLE skus (
    id VARCHAR(50) PRIMARY KEY,        -- ID SKU из XLSX файла импорта
    manufacturer_id INT NOT NULL REFERENCES manufacturers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(100) NOT NULL,
    category VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. УПРОЩЕННАЯ МАТРИЦА (ПЛОСКАЯ СТРУКТУРА БЕЗ ЛИШНИХ ТАБЛИЦ И ФЛАГОВ)
-- Наличие записи = чекбокс установлен. Отсутствие записи = снят.
CREATE TABLE store_matrices (
    manufacturer_id INT REFERENCES manufacturers(id) ON DELETE CASCADE,
    sku_id VARCHAR(50) REFERENCES skus(id) ON DELETE CASCADE,
    retail_network VARCHAR(100) NOT NULL,
    store_format VARCHAR(50) NOT NULL,
    PRIMARY KEY (manufacturer_id, sku_id, retail_network, store_format)
);

-- 8. АНКЕТЫ И ИХ ВЕРСИИ ДЛЯ ИСТОРИЧНОСТИ ОТЧЕТОВ
CREATE TABLE questionnaires (
    id SERIAL PRIMARY KEY,
    manufacturer_id INT NOT NULL REFERENCES manufacturers(id) ON DELETE RESTRICT,
    version INT NOT NULL DEFAULT 1,
    is_auto_clone_enabled BOOLEAN NOT NULL DEFAULT false, -- <--- ДОБАВЛЕНО ДЛЯ АВТОПОДСТАНОВКИ
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (manufacturer_id, version)
);


-- 9. ТАБЛИЦА ВОПРОСОВ АНКЕТЫ (ЧЕСТНОЕ РЕКУРСИВНОЕ ДЕРЕВО УСЛОВИЙ С ВЕТВЛЕНИЕМ)
CREATE TABLE questionnaire_questions (
    id SERIAL PRIMARY KEY,
    questionnaire_id INT NOT NULL REFERENCES questionnaires(id) ON DELETE CASCADE,
    parent_question_id INT NULL REFERENCES questionnaire_questions(id) ON DELETE CASCADE, -- Родитель для вложенных
    section VARCHAR(50) NOT NULL CHECK (section IN ('general', 'sku')),
    sku_id VARCHAR(50) NULL REFERENCES skus(id) ON DELETE CASCADE, -- Заполнен только для секции SKU
    question_text TEXT NOT NULL,
    -- response_type VARCHAR(50) NOT NULL CHECK (response_type IN ('text', 'number', 'photo', 'condition', 'select', 'date')),
    response_type VARCHAR(50) NOT NULL CHECK (response_type IN ('text', 'number', 'photo', 'condition', 'select', 'date', 'multiselect')),
    -- CONSTRAINT questionnaire_questions_response_type_check CHECK (response_type IN ('text', 'number', 'photo', 'condition', 'select', 'date'))
    sort_order INT NOT NULL DEFAULT 0,
    
    -- Логика ветвления дерева:
    -- NULL  = Обычный вопрос верхнего уровня
    -- true  = Дочерний вопрос для ветки "Условие выполнено" (Да)
    -- false = Дочерний вопрос для ветки "Условие не выполнено" (Нет)
    condition_branch BOOLEAN NULL,
    
    -- Параметры оператора сравнения (Условия по тексту 1/тексту 2, числам, датам, логические И/ИЛИ)
    condition_operator VARCHAR(50) NULL,
    condition_value_text TEXT[] NULL,       -- Массив строк для текстов или дат
    condition_value_numeric NUMERIC(12,2) NULL, -- Число для сравнения (больше/меньше/равно)    
    available_options TEXT[] NULL, -- ДОБАВИТЬ ЭТУ СТРОКУ (Варианты ответов для списков выбора)
    allow_gallery BOOLEAN NOT NULL DEFAULT false, -- Промышленный флаг управления доступом к галерее S3
    
        -- Чистые поля для абстрактной межвопросовой зависимости (Кейс с Промо-вопросом) БЕЗ ГВОЗДЕЙ
    depends_on_question_id INT NULL REFERENCES questionnaire_questions(id) ON DELETE SET NULL,
    depends_on_operator VARCHAR(50) NULL, -- Операторы: =, <=, >=, >, <, !=
    depends_on_value TEXT NULL
);




-- 10. СУЩНОСТЬ «ВИЗИТ» (ПЕРВИЧНЫЙ PK — UUID ДЛЯ НАДЕЖНОГО OFFLINE-FIRST НА ANDROID)
CREATE TABLE visits (
    id UUID PRIMARY KEY,               -- UUID генерируется на Android-девайсе в оффлайне
    store_id_1c VARCHAR(50) NOT NULL REFERENCES stores(id_1c) ON DELETE RESTRICT,
    merchandiser_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE NULL,
    is_scheduled BOOLEAN NOT NULL,     -- По маршруту (Да/Нет)
    is_online BOOLEAN NOT NULL,        -- Режим старта визита (Онлайн/Оффлайн)
    start_latitude NUMERIC(9,6) NULL,
    start_longitude NUMERIC(9,6) NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'finished', 'auto_finished')),
    finish_type VARCHAR(50) NULL CHECK (finish_type IN ('normal', 'auto')), -- Ручное завершение или через 24 часа
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 11. ОТЧЕТ ПО КОНКРЕТНОМУ ПРОИЗВОДИТЕЛЮ ВНУТРИ ВИЗИТА (Без принудительных флагов блокировки)
CREATE TABLE visit_manufacturer_reports (
    id SERIAL PRIMARY KEY,
    visit_id UUID NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    manufacturer_id INT NOT NULL REFERENCES manufacturers(id) ON DELETE RESTRICT,
    questionnaire_id INT NOT NULL REFERENCES questionnaires(id) ON DELETE RESTRICT, -- Жесткая фиксация исторической версии анкеты
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE NULL
);

-- 12. ОТВЕТЫ НА ВОПРОСЫ АНКЕТЫ
CREATE TABLE report_answers (
    id SERIAL PRIMARY KEY,
    report_id INT NOT NULL REFERENCES visit_manufacturer_reports(id) ON DELETE CASCADE,
    question_id INT NOT NULL REFERENCES questionnaire_questions(id) ON DELETE RESTRICT,
    sku_id VARCHAR(50) NULL REFERENCES skus(id) ON DELETE RESTRICT, -- Заполнено только для SKU-секций
    value_text TEXT NULL,
    value_number NUMERIC(12,2) NULL,    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 13. ССЫЛКИ НА ФОТОГРАФИИ В S3 (ПРИВЯЗАНЫ НАПРЯМУЮ К ОТВЕТАМ ДЛЯ ОДНОЗНАЧНОЙ ОТЧЕТНОСТИ)
CREATE TABLE report_photos (
    id SERIAL PRIMARY KEY,
    answer_id INT NOT NULL REFERENCES report_answers(id) ON DELETE CASCADE,
    s3_key TEXT NOT NULL,              -- Уникальный ключ объекта внутри бакета S3
    s3_url TEXT NOT NULL,              -- Ссылка на скачивание фото
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 14. СИСТЕМНЫЕ ИНДЕКСЫ ДЛЯ ОПТИМИЗАЦИИ ТЯЖЕЛЫХ И ЧАСТЫХ ВЫГРУЗОК ОТЧЕТОВ
-- ============================================================================
CREATE INDEX idx_users_role_phone ON users(role, phone);
CREATE INDEX idx_stores_supervisor_me ON stores(supervisor_id, merchandiser_id);
CREATE INDEX idx_stores_reporting ON stores(region, retail_network, store_format);
CREATE INDEX idx_skus_manufacturer_brand ON skus(manufacturer_id, brand, category);
CREATE INDEX idx_question_questions_lookup ON questionnaire_questions(questionnaire_id, parent_question_id);
CREATE INDEX idx_visits_reporting_dates ON visits(started_at, store_id_1c, merchandiser_id);
CREATE INDEX idx_vmr_visit_manufacture ON visit_manufacturer_reports(visit_id, manufacturer_id);
CREATE INDEX idx_report_answers_main ON report_answers(report_id, question_id, sku_id);
CREATE INDEX idx_report_photos_answer ON report_photos(answer_id);
