# 🚀 Oxidized Router.db Web Manager

Веб-інтерфейс на **Go (Golang)** для зручного та безпечного редагування бази обладнання `router.db` програми [Oxidized](https://github.com/ytti/oxidized), керування записами та перезапуску системного сервісу `oxidized.service` під Linux.

---

## ✨ Основні можливості

1. **Повна сумісність з Linux**: Збирається в один автономний бінарний файл без зовнішніх залежностей (весь HTML/CSS/JS вбудований через `go:embed`).
2. **Конфігурація через `config.yaml`**:
   - Шлях до файлу бази (за замовчуванням `/var/lib/oxidized/router.db` або `./router.db`).
   - Порт веб-консолі (наприклад, `8080`).
   - Команди перезапуску та перевірки статусу `oxidized.service`.
   - Логін та bcrypt-хешований пароль.
3. **Формат записів `router.db`**:
   ```text
   name:ip:model:username:password:group:port
   ```
   - Поле `port` може залишатися порожнім (підтримується як `name:ip:...:group:`, так і `name:ip:...:group:22`).
   - Зберігаються коментарі файлу (рядки, що починаються з `#`).
   - Автоматичне створення резервної копії `.bak` перед кожним збереженням.
4. **Керування обладнанням (CRUD)**:
   - Додавання нового пристрою (з валідацією та живим попереднім переглядом рядка).
   - Редагування існуючих записів.
   - Клонування/дублювання пристроїв в один клік.
   - Видалення записів із модальним підтвердженням.
   - Миттєвий пошук та фільтрація за назвою, IP, моделлю та групою.
   - Копіювання IP-адреси в буфер обміну в один клік.
   - Приховування/показ паролів перемикачем (Eye icon).
   - Вбудований **Raw-редактор** для прямого перегляду/редагування файлу `router.db`.
5. **Керування сервісом `oxidized.service`**:
   - Кнопка перезавантаження сервісу з модальним вікном, термінальним журналом виконання та перевіркою статусу (`active`/`inactive`).
6. **Безпечна авторизація**:
   - Вхід за логіном і паролем (за замовчуванням: `admin` / `admin`).
   - Пароль зберігається у вигляді криптографічного bcrypt-хешу в `config.yaml`.
   - Можливість змінити пароль безпосередньо з веб-інтерфейсу (автоматично оновлює `config.yaml`).
   - Захищені HTTP-only cookie сесії (термін дії 24 години).

---

## 📁 Структура проєкту

```text
oxidized_edit/
├── config.yaml               # Файл налаштувань (порт, шлях до router.db, пароль)
├── router.db                 # Зразок бази пристроїв Oxidized
├── main.go                   # Точка входу, запуск HTTP-сервера
├── Makefile                  # Команди збірки для Linux та Windows
├── oxidized-web.service      # Готовий systemd unit файл для Linux
├── oxidized-manager-linux    # Готовий скомпільований бінарник під Linux (amd64)
├── internal/
│   ├── config/               # Завантаження/збереження YAML та bcrypt хешування
│   ├── database/             # Парсер, валідатор, CRUD та бекап router.db
│   ├── service/              # Перезапуск і перевірка статусу oxidized.service
│   ├── auth/                 # Сесії, авторизаційні токени та cookie
│   └── handlers/             # REST API ендпоінти
└── web/
    ├── embed.go              # Вбудовування ресурсів у бінарний файл
    └── static/
        ├── index.html        # SPA інтерфейс
        ├── styles.css        # Преміальна темна тема, скло (glassmorphism)
        └── app.js            # Інтерактивна логіка та взаємодія з API
```

---

## ⚙️ Налаштування (`config.yaml`)

```yaml
server:
  host: 0.0.0.0
  port: 8080 # Порт на якому працює веб-консоль

oxidized:
  # Шлях до файлу router.db на Linux сервері:
  db_path: "/var/lib/oxidized/router.db"

  # Команда для перезапуску сервісу:
  restart_command: "systemctl restart oxidized.service"

  # Команда перевірки статусу:
  status_command: "systemctl is-active oxidized.service"

  # Автоматичне створення резервної копії (.bak):
  auto_backup: true

auth:
  username: "admin"
  # Хеш пароля (за замовчуванням: admin)
  password_hash: "$2a$10$f4Xc1J9nr1W29G//Qtg9v.CLDC1Wv0rFNMktOU8V.Tf/cNA3ZrgXG"
  session_secret: "oxidized-manager-secure-session-key-2026"
```

> **Порада**: Якщо ви бажаєте встановити простий пароль вручну у файлі, замість `password_hash` вкажіть:
> ```yaml
> auth:
>   username: "admin"
>   password: "ВашПароль"
> ```
> При першому запуску програма автоматично перетворить його на безпечний bcrypt-хеш та прибере відкритий текст!

---

## 🛠️ Збірка

### Збірка для Linux (на Linux або через крос-компіляцію з Windows):
```bash
# Через Makefile:
make build-linux

# Або вручну:
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o oxidized-manager-linux .
```

### Запуск тестів:
```bash
make test
```

---

## 🐧 Встановлення та запуск на Linux

### 1. Копіювання файлів на сервер
```bash
# Створіть директорію для програми
sudo mkdir -p /opt/oxidized-manager

# Скопіюйте бінарник і конфіг
sudo cp oxidized-manager-linux /opt/oxidized-manager/oxidized-manager
sudo cp config.yaml /opt/oxidized-manager/config.yaml
sudo chmod +x /opt/oxidized-manager/oxidized-manager
```

### 2. Налаштування прав для перезапуску `oxidized.service`
Якщо програма запускається під окремим користувачем (наприклад, `oxidized`), дозвольте йому перезапускати сервіс без введення пароля sudo:
Створіть файл `/etc/sudoers.d/oxidized-manager`:
```sudoers
oxidized ALL=(ALL) NOPASSWD: /bin/systemctl restart oxidized.service, /bin/systemctl is-active oxidized.service
```
І в `config.yaml` вкажіть:
```yaml
restart_command: "sudo systemctl restart oxidized.service"
```

### 3. Налаштування systemd сервісу
Скопіюйте готовий файл `oxidized-web.service`:
```bash
sudo cp oxidized-web.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now oxidized-web.service
```

Перевірте статус:
```bash
sudo systemctl status oxidized-web.service
```

### 4. Вхід у веб-інтерфейс
Відкрийте в браузері:
```
http://IP_ВАШОГО_СЕРВЕРА:8080
```
- **Логін**: `admin`
- **Пароль**: `admin`
*(після першого входу рекомендується змінити пароль у правому верхньому кутку)*
