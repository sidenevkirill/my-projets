<script src="https://vk.com/js/api/xd_connection.js?2"></script>
<script>
// Инициализация VK Mini App
VK.init(function() {
    console.log('VK Mini App initialized');
    
    // Получаем информацию о пользователе
    VK.api('users.get', {fields: 'photo_100'}, function(data) {
        if (data.response) {
            const user = data.response[0];
            // Обновляем профиль
            document.getElementById('userName').textContent = user.first_name + ' ' + user.last_name;
        }
    });
    
    // Получаем токен доступа
    VK.Auth.getLoginStatus(function(response) {
        if (response.session) {
            const token = response.session.mid;
            // Отправляем токен на сервер
            fetch('/auth/vk-mini-app?vk_access_token=' + token);
        }
    });
});
</script>
