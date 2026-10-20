-- reel.lua - журнал событий для RP Reel (FiveM)
--
-- Сделай resource: folder reel/ с этим файлом и fxmanifest.lua,
-- ресурс start'уется автоматически, пишет rpreel.jsonl
-- в корень сервера. Серверные события:
--   exports['reel']:reel('kill', 'Nick', 'Victim')
--   exports['reel']:reel('custom', 'админ', 'начало ивента')

local FILE = 'rpreel.jsonl'

local function reel(kind, player, other)
	local line = string.format(
		'{"ts":%d,"type":"%s","player":"%s","other":"%s"}\n',
		os.time(), kind, player or '', other or '')
	local f = io.open(FILE, 'a')
	if f then
		f:write(line)
		f:close()
	end
end

exports('reel', reel)

CreateThread(function()
	-- готовые подписки, удали лишнее
	AddEventHandler('playerDied', function(ped, killer)
		local p = source
		reel('death', GetPlayerName(p) or '?')
	end)
	AddEventHandler('playerConnecting', function(name)
		reel('join', name)
	end)
	AddEventHandler('playerDropped', function(reason)
		reel('leave', GetPlayerName(source) or '?')
	end)
end)

-- типы: kill, death, wreck, arrest, rob, admin, join, leave, custom
