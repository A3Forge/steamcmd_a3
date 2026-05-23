# steamcmd

Go-пакет — обёртка над SteamCMD для управления Arma 3 Dedicated Server и Workshop-модами.

Один экземпляр `SteamCmd` запускает не более одного процесса `steamcmd.exe` одновременно. Попытка запустить вторую операцию вернёт `ErrSteamCmdAlreadyActive`.

## Возможности

- автоматическая загрузка `steamcmd.exe`, если он ещё не установлен
- хранение нескольких Steam-аккаунтов, выбор активного
- логин с поддержкой Steam Guard
- установка/обновление Arma 3 Dedicated Server (`app_update 233780`)
- установка/обновление Workshop-модов по ID (`workshop_download_item`)
- список установленных модов и путь до папки с ними
- парсинг прогресса загрузки из stdout SteamCMD (поле `Progress float64`)
- колбэк `OnStatusChange` — вызывается при каждом изменении статуса или прогресса
- принудительное завершение дочернего процесса через `Close()`

## Пример

Готовый HTTP-сервер с веб-интерфейсом находится в [`cmd/example`](cmd/example):

```bash
cd cmd/example
go run .
# → http://localhost:8080
```

## Использование пакета

### Инициализация

```go
import steamcmd "github.com/A3Forge/steamcmd_a3"

scmd := steamcmd.NewSteamCmd()
defer scmd.Close()

// Рабочая директория SteamCMD (по умолчанию — текущая)
scmd.SetSteamCmdDir("C:/steamcmd")

// Колбэк при изменении статуса или прогресса
scmd.OnStatusChange = func(s steamcmd.SteamCmdStatus) {
    fmt.Println(s.Status, s.Progress)
}
```

### Аккаунты

```go
id, err := scmd.AddUser("username", "password")
err  = scmd.SetActiveUser(id)
err  = scmd.DeleteUser(id)

users  := scmd.AllUsers()
active := scmd.GetActiveUser()
```

Аккаунты сохраняются в `%LOCALAPPDATA%\steamcmd_wr\credentials.json`.  
Пароли хранятся в plain text — не использовать на публичном сервере без шифрования.

### Логин

```go
err := scmd.TryLogin()
if err != nil && err.Error() == steamcmd.STATUS_STEAMGUARD_CODE {
    err = scmd.TryLoginWithSteamGuardCode("XXXXX")
}
```

### Установка/обновление сервера

Запускается асинхронно. Завершение отслеживается через `Status()` или `OnStatusChange`.

```go
err := scmd.ValidateArma3Server("C:/arma3server")
```

### Установка/обновление модов

Моды скачиваются последовательно, по одному.

```go
err := scmd.ValidateMods([]string{"450814997", "463939057"})

dir  := scmd.ModsDir()       // абсолютный путь до папки с модами
mods, err := scmd.InstalledMods() // [{ID, Path}]
```

### Статус и прогресс

```go
s := scmd.Status()
s.Status        // строка-константа, "" означает простой
s.StatusDetails // детали (например, список ID модов)
s.Progress      // float64, 0–100
```

### Статусы

```go
STATUS_FINE                         // "" — операция завершена / простой
STATUS_STEAMGUARD_CODE              // требуется код Steam Guard
STATUS_INVALID_USER                 // неверный логин или пароль
STATUS_STEAMCMD_RATELIMIT           // Steam вернул rate limit
STATUS_STEAMCMD_INSTALLING          // идёт загрузка steamcmd.exe
STATUS_STEAMCMD_ERROR_DOWNLOAD      // ошибка загрузки steamcmd.exe
STATUS_STEAMCMD_CANT_OPEN_ZIP       // ошибка открытия архива установщика
STATUS_STEAMCMD_CANT_WRITE          // ошибка записи файлов
STATUS_STEAMCMD_INTALLING_A3SERVER  // идёт установка/обновление сервера
STATUS_STEAMCMD_MODS_VALIDATING     // идёт установка/обновление модов
STATUS_STEAMCMD_MODS_VALIDATING_ERR // ошибка установки модов
STATUS_STEAMCMD_MISSING_PARAMS      // SteamCMD получил неверные аргументы
STATUS_INIT_ERR                     // ошибка инициализации
```

### Завершение

```go
defer scmd.Close()       // завершить steamcmd.exe при выходе
scmd.CloseOnInterrupt()  // завершить по Ctrl+C (запускает горутину)
```

## Логи SteamCMD

```
steamcmd_wr/logs/console_log.txt
steamcmd_wr/logs/connection_log.txt
steamcmd_wr/logs/content_log.txt
```

Если лог нельзя удалить — `steamcmd.exe` ещё запущен. Вызови `scmd.Close()`.
