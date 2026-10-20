// reel.js - журнал событий для RP Reel (RAGE MP, сервер на JS)
//
// Положи рядом с серверным скриптом и трогай в нужных местах:
//   const { reel } = require('./reel');
//   reel('kill', killer.name, victim.name);
//
// Или подпишись на готовые события ниже - они уже включены.
// Файл rpreel.jsonl появится рядом с серверным процессом.

const fs = require('fs');

const FILE = 'rpreel.jsonl';

function reel(type, player, other) {
	const line = JSON.stringify({
		ts: Math.floor(Date.now() / 1000),
		type: type,
		player: player || '',
		other: other || '',
	}) + '\n';
	try {
		fs.appendFileSync(FILE, line);
	} catch (e) {
		console.warn('[reel] не записать журнал:', e.message);
	}
}

module.exports = { reel };

// --- готовые подписки, удали лишнее ---

mp.events.add({
	playerDeath(player, killer) {
		reel('kill', killer && killer.name, player.name);
	},
	playerQuit(player) {
		reel('leave', player.name);
	},
	playerJoin(player) {
		reel('join', player.name);
	},
	playerVehicleDeath(player, vehicle) {
		reel('wreck', player.name);
	},
});

// Свое событие: reel('custom', 'админ', 'перезапуск сервера в 5 минут');
// Типы: kill, death, wreck, arrest, rob, admin, join, leave, custom.
// Любой свой тип тоже выведется на таймлайн.
