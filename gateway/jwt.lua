local M = {}
local json = require "cjson"

local function base64url_decode(input)
    local remainder = #input % 4
    if remainder > 0 then
        input = input .. string.rep('=', 4 - remainder)
    end
    input = string.gsub(input, '-', '+')
    input = string.gsub(input, '_', '/')
    return ngx.decode_base64(input)
end

function M:verify(secret, token)
    ngx.log(ngx.INFO, "JWT verify called with token length: ", token and #token or 0)

    if not token then
        return {valid = false, reason = "no token"}
    end

    local parts = {}
    for part in string.gmatch(token, "([^%.]+)") do
        table.insert(parts, part)
    end

    if #parts ~= 3 then
        ngx.log(ngx.ERR, "JWT: Invalid format, parts count: ", #parts)
        return {valid = false, reason = "invalid format"}
    end

    local header_part, payload_part, signature_part = parts[1], parts[2], parts[3]

    -- Декодируем header и payload
    local header_json = base64url_decode(header_part)
    local payload_json = base64url_decode(payload_part)

    if not header_json or not payload_json then
        ngx.log(ngx.ERR, "JWT: Decode error")
        return {valid = false, reason = "decode error"}
    end

    local header_ok, header = pcall(json.decode, header_json)
    local payload_ok, payload = pcall(json.decode, payload_json)

    if not header_ok or not payload_ok then
        ngx.log(ngx.ERR, "JWT: JSON parse error")
        return {valid = false, reason = "json parse error"}
    end

    ngx.log(ngx.INFO, "JWT: Header decoded: ", header_json)
    ngx.log(ngx.INFO, "JWT: Payload decoded: ", payload_json)

    -- Проверяем алгоритм
    if header.alg ~= "HS256" then
        ngx.log(ngx.ERR, "JWT: Unsupported algorithm: ", header.alg or "nil")
        return {valid = false, reason = "unsupported algorithm"}
    end

    -- Проверяем срок действия
    local now = ngx.time()
    if payload.exp and payload.exp < now then
        ngx.log(ngx.ERR, "JWT: Token expired. Now: ", now, ", Exp: ", payload.exp)
        return {valid = false, reason = "token expired"}
    end

    ngx.log(ngx.INFO, "JWT: Token is valid, login: ", payload.login or "unknown")
    return {
        valid = true,
        header = header,
        payload = payload
    }
end

return M