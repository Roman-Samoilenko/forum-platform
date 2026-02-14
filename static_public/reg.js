document.addEventListener('DOMContentLoaded', function() {
    const registrationForm = document.getElementById('registrationForm');

    registrationForm.addEventListener('submit', async function(e) {
        e.preventDefault();

        const data = {
            login: document.getElementById('login').value,
            password: document.getElementById('password').value
        };

        try {
            const response = await fetch('/submit-form', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });

            if (response.ok) {
                const result = await response.json();
                showAlert('success', result.message || 'Вход успешен!');
                setTimeout(() => {
                    window.location.href = '/';
                }, 1500);
            } else {
                const error = await response.json();
                showAlert('error', error.message || 'Ошибка регистрации');
            }
        } catch (error) {
            showAlert('error', 'Ошибка сети');
            console.error('Ошибка:', error);
        }
    });

    function showAlert(type, message) {
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert ${type}`;
        alertDiv.textContent = message;

        document.body.appendChild(alertDiv);

        setTimeout(() => {
            alertDiv.remove();
        }, 3000);
    }

    const inputs = document.querySelectorAll('input');

    inputs.forEach(input => {

        input.addEventListener('input', function() {
            if (this.value.length > 0) {
                this.style.boxShadow = '0 0 15px rgba(0, 255, 0, 0.3)';
            } else {
                this.style.boxShadow = '';
            }
        });
    });

    const submitBtn = document.querySelector('.submit-btn');
    submitBtn.addEventListener('click', function() {
        this.style.transform = 'scale(0.98)';
        setTimeout(() => {
            this.style.transform = '';
        }, 150);
    });
});