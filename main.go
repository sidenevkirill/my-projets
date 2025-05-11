package main

import (
    "fmt"
    "net/http"
    "os"
    "sync"
)

var (
    users      = make(map[string]string) // Хранилище пользователей (username: password)
    usersMutex sync.Mutex                 // Мьютекс для безопасного доступа к хранилищу
)

func main() {
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/register", registerHandler)
    http.HandleFunc("/login", loginHandler)

    fmt.Println("Сервер запущен на http://localhost:8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println(err)
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "<h1>Добро пожаловать!</h1>")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        username := r.FormValue("username")
        password := r.FormValue("password")

        usersMutex.Lock()
        defer usersMutex.Unlock()

        if _, exists := users[username]; exists {
            http.Error(w, "Пользователь уже существует", http.StatusConflict)
            return
        }

        // Сохраняем пользователя в памяти
        users[username] = password

        // Сохраняем пользователя в файл
        err := saveUserToFile(username, password)
        if err != nil {
            http.Error(w, "Ошибка при сохранении пользователя", http.StatusInternalServerError)
            return
        }

        fmt.Fprintf(w, "<h1>Регистрация успешна!</h1>")
        return
    }

    // Отображение формы регистрации
    tmpl := `
        <html>
        <head>
            <title>Регистрация</title>
            <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css">
            <style>
                body {
                    font-family: Arial, sans-serif;
                    background-color: #f2f3f5;
                    margin: 0;
                    padding: 0;
                }
                .container {
                    max-width: 400px;
                    margin: 100px auto;
                    background-color: white;
                    border-radius: 8px;
                    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
                    padding: 20px;
                }
                h1 {
                    text-align: center;
                    color: #4a4a4a;
                }
                label {
                    display: block;
                    margin-bottom: 5px;
                }
                input[type="text"],
                input[type="password"] {
                    width: calc(100% - 20px);
                    padding: 10px;
                    margin-bottom: 15px;
                    border-radius: 5px;
                    border: 1px solid #dcdcdc; 
                }
                button {
                    background-color: #007bff; 
                    color: white; 
                    border: none; 
                    padding: 10px; 
                    border-radius: 5px; 
                    cursor: pointer; 
                    width: 100%;
                }
                button:hover { 
                   background-color:#0056b3; 
                   transition: background-color 0.3s ease; 
               } 
               .footer {
                   text-align:center; 
                   margin-top :15 px; 
               }  
           </style> 
       </head> 
       <body> 
           <div class="container">
               <h1>Регистрация</h1>
               <form method="POST"> 
                   <label for="username">Имя пользователя:</label> 
                   <input type="text" id="username" name="username" required> 

                   <label for="password">Пароль:</label> 
                   <input type="password" id="password" name="password" required>

                   <button type="submit">Зарегистрироваться</button> 
               </form>
               <div class="footer">
                   <p>Уже есть аккаунт? <a href="/login">Войти</a></p>
               </div>
           </div>
       </body> 
       </html>`
    
   w.Write([]byte(tmpl))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        username := r.FormValue("username")
        password := r.FormValue("password")

        usersMutex.Lock()
        defer usersMutex.Unlock()

        if storedPassword, exists := users[username]; exists && storedPassword == password {
            fmt.Fprintf(w, "<h1>Добро пожаловать, %s!</h1>", username)
            return
        }

        http.Error(w, "Неверное имя пользователя или пароль", http.StatusUnauthorized)
        return
    }

    // Отображение формы входа с тем же дизайном
    tmpl := `
        <html>
        <head>
            <title>Вход</title>
            <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css">
          <style>
                body {
                    font-family: Arial, sans-serif;
                    background-color: #f2f3f5;
                    margin: 0;
                    padding: 0;
                }
                .container {
                    max-width: 400px;
                    margin: 100px auto;
                    background-color: white;
                    border-radius: 8px;
                    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
                    padding: 20px;
                }
                h1 {
                    text-align: center;
                    color: #4a4a4a;
                }
                label {
                    display: block;
                    margin-bottom: 5px;
                }
                input[type="text"],
                input[type="password"] {
                    width: calc(100% - 20px);
                    padding: 10px;
                    margin-bottom: 15px;
                    border-radius: 5px;
                    border: 1px solid #dcdcdc; 
                }
                button {
                    background-color: #007bff; 
                    color: white; 
                    border: none; 
                    padding: 10px; 
                    border-radius: 5px; 
                    cursor: pointer; 
                    width: 100%;
                }
                button:hover { 
                   background-color:#0056b3; 
                   transition: background-color 0.3s ease; 
               } 
               .footer {
                   text-align:center; 
                   margin-top :15 px; 
               }  
           </style> 
         </head>     
         <body>     
             <div class="container">     
                 <h1>Вход</h1>     
                 <form method="POST">     
                     <label for="username">Имя пользователя:</label>     
                     <input type="text" id="username" name="username" required>

                     <label for="password">Пароль:</label>
                     <input type="password" id="password" name="password" required>

                     <button type="submit">Войти</button>
                 </form>
                 <div class="footer">
                   <p>Нет аккаунта? <a href="/register">Зарегистрироваться</a></p>
               </div>
             </div>
         </body>
         </html>`
    
   w.Write([]byte(tmpl))
}

// Функция для сохранения пользователя в файл
func saveUserToFile(username string, password string) error {
    file, err := os.OpenFile("users.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(fmt.Sprintf("%s:%s\n", username, password))
    return err
}