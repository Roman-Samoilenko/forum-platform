local M = {}

local jwt_secret = "hfdjhj23"

function M.get_token_from_cookie()
    local cookie_header = ngx.var.http_cookie
    if not cookie_header then
        ngx.log(ngx.INFO, "No cookie header found")
        return nil
    end

    ngx.log(ngx.INFO, "Cookie header found: ", cookie_header)

    -- Парсинг cookie для поиска auth_token
    for cookie in string.gmatch(cookie_header, "([^;]+)") do
        local name, value = string.match(cookie, "^%s*([^=]+)=(.*)$")
        if name then
            local clean_name = string.gsub(name, "%s+", "")
            ngx.log(ngx.INFO, "Found cookie: '", clean_name, "' = '", (value and string.sub(value, 1, 20) .. "..." or "nil"), "'")

            if clean_name == "auth_token" then
                ngx.log(ngx.INFO, "Found auth_token")
                return value
            end
        end
    end

    ngx.log(ngx.INFO, "No auth_token found in cookies")
    return nil
end

-- Функция для проверки валидности JWT
function M.verify_jwt(token)
    ngx.log(ngx.INFO, "verify_jwt function called with token length: ", token and #token or 0)

    if not token then
        ngx.log(ngx.INFO, "JWT token is missing")
        return false, "Token missing"
    end

    -- Загружаем модуль JWT
    local jwt_ok, jwt = pcall(require, "resty.jwt")
    if not jwt_ok then
        ngx.log(ngx.ERR, "Failed to load resty.jwt module: ", jwt)
        return false, "JWT module not available"
    end

    ngx.log(ngx.INFO, "JWT module loaded successfully")

    -- Проверяем токен
    local jwt_obj = jwt:verify(jwt_secret, token)

    if not jwt_obj then
        ngx.log(ngx.ERR, "JWT verification returned nil")
        return false, "JWT verification failed"
    end

    ngx.log(ngx.INFO, "JWT verification result: valid=", tostring(jwt_obj.valid), ", reason=", jwt_obj.reason or "none")

    if not jwt_obj.valid then
        ngx.log(ngx.ERR, "JWT verification failed: ", jwt_obj.reason or "unknown reason")
        return false, jwt_obj.reason or "invalid token"
    end

    -- Извлекаем логин из payload
    local login = "unknown"
    if jwt_obj.payload and jwt_obj.payload.login then
        login = jwt_obj.payload.login
        ngx.log(ngx.INFO, "Extracted login: ", login)
    else
        ngx.log(ngx.WARN, "No login found in JWT payload")
    end

    ngx.log(ngx.INFO, "JWT token is valid for user: ", login)
    return true, {login = login, payload = jwt_obj.payload}
end

-- Основная функция аутентификации
function M.authenticate()
    ngx.log(ngx.INFO, "=== Starting JWT authentication ===")

    -- Инициализируем переменные
    ngx.var.jwt_valid = "false"
    ngx.var.jwt_user_login = ""
    ngx.var.jwt_error = ""

    local token = M.get_token_from_cookie()
    if not token then
        ngx.log(ngx.INFO, "No token found, authentication failed")
        ngx.var.jwt_error = "No token"
        ngx.log(ngx.INFO, "Final variables: jwt_valid=", ngx.var.jwt_valid, ", jwt_user_login=", ngx.var.jwt_user_login)
        return false
    end

    ngx.log(ngx.INFO, "Token found, length: ", #token)
    local is_valid, result = M.verify_jwt(token)

    if is_valid then
        ngx.var.jwt_valid = "true"
        ngx.var.jwt_user_login = result.login or ""
        ngx.log(ngx.INFO, "=== Authentication SUCCESSFUL. User: ", result.login or "unknown")
        ngx.log(ngx.INFO, "Final variables: jwt_valid=", ngx.var.jwt_valid, ", jwt_user_login=", ngx.var.jwt_user_login)
        return true
    else
        ngx.var.jwt_error = tostring(result)
        ngx.log(ngx.INFO, "=== Authentication FAILED. Reason: ", tostring(result))
        ngx.log(ngx.INFO, "Final variables: jwt_valid=", ngx.var.jwt_valid, ", jwt_user_login=", ngx.var.jwt_user_login)
        return false
    end
end

return M