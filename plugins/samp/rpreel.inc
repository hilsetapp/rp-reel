/*
	rpreel.inc - журнал событий для RP Reel
	=====================================

	Что делает: пишет в rpreel.jsonl каждое событие с точным
	временем. RP Reel потом кладет эти события на таймлайн
	видеозаписи и вырезает клипы.

	Установка (SA-MP, Pawn):
		1. скопируй rpreel.inc в pawno/include
		2. в гамемоде: #include <rpreel>
		3. в OnGameModeInit:  Reel_Open();
		   в OnGameModeExit: Reel_Close();
		4. в нужных местах вызывай Reel_Log(...)
		   или готовые Reel_Kill / Reel_Wreck / Reel_Arrest

	Запись видео: OBS на той же машине, что и сервер,
	часы совпадают. Если сервер и запись на разных машинах,
	укажи сдвиг в приложении.
*/

#if defined _rpreel_included
	#endinput
#endif
#define _rpreel_included

static Reel_Handle = 0;

/* Открывает журнал. Сам вызовется в Reel_Log, если забыл. */
stock Reel_Open()
{
	if (!Reel_Handle) {
		Reel_Handle = fopen("rpreel.jsonl", io_append);
	}
	return Reel_Handle;
}

/* Закрывает журнал, например в OnGameModeExit. */
stock Reel_Close()
{
	if (Reel_Handle) {
		fclose(Reel_Handle);
		Reel_Handle = 0;
	}
}

/* Пишет событие. type: kill, death, wreck, arrest, rob, admin, join, leave, custom. */
stock Reel_Log(const type[], const player[], const other[] = "")
{
	if (!Reel_Open()) return 0;

	new line[256];
	format(line, sizeof line,
		"{\"ts\":%d,\"type\":\"%s\",\"player\":\"%s\",\"other\":\"%s\"}\n",
		gettime(), type, player, other);
	fwrite(Reel_Handle, line);
	return 1;
}

/* Удобные обертки: передавай ид игрока, имя возьмется сам. */
stock Reel_Name(playerid, buffer[], size)
{
	if (playerid < 0) {
		buffer[0] = EOS;
		return 0;
	}
	return GetPlayerName(playerid, buffer, size);
}

stock Reel_Kill(killerid, victimid, weaponid)
{
	new k[MAX_PLAYER_NAME], v[MAX_PLAYER_NAME];
	Reel_Name(killerid, k, sizeof k);
	Reel_Name(victimid, v, sizeof v);
	Reel_Log("kill", k, v);
}

stock Reel_Death(playerid)
{
	new p[MAX_PLAYER_NAME];
	Reel_Name(playerid, p, sizeof p);
	Reel_Log("death", p);
}

stock Reel_Wreck(playerid)
{
	new p[MAX_PLAYER_NAME];
	Reel_Name(playerid, p, sizeof p);
	Reel_Log("wreck", p);
}

stock Reel_Arrest(copid, suspectid)
{
	new c[MAX_PLAYER_NAME], s[MAX_PLAYER_NAME];
	Reel_Name(copid, c, sizeof c);
	Reel_Name(suspectid, s, sizeof s);
	Reel_Log("arrest", c, s);
}

stock Reel_Custom(const label[], const player[] = "")
{
	Reel_Log("custom", player, label);
}

/* Примеры вызовов в гамемоде:

public OnPlayerDeath(playerid, killerid, reason)
{
	if (killerid != INVALID_PLAYER_ID) {
		Reel_Kill(killerid, playerid, reason);
	} else {
		Reel_Death(playerid);
	}
	return 1;
}

public OnPlayerConnect(playerid)
{
	new p[MAX_PLAYER_NAME];
	GetPlayerName(playerid, p, sizeof p);
	Reel_Log("join", p);
	return 1;
}

public OnPlayerDisconnect(playerid, reason)
{
	new p[MAX_PLAYER_NAME];
	GetPlayerName(playerid, p, sizeof p);
	Reel_Log("leave", p);
	return 1;
}
*/
