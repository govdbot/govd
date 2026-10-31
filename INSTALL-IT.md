# Installare questa versione di Govd

Questa versione funziona indipendentemente dall'accettazione delle modifiche da parte di govdbot/govd. Non esiste un comando `sudo install` dedicato e non pubblichiamo ancora un'immagine pronta: si compila dai sorgenti. Non eseguire il vecchio `docker-compose.yaml`: scaricherebbe l'immagine originale.

## Nuova installazione

Procedura per Linux amd64 con Git, curl e Docker Engine con Compose v2 gia installati. Altre architetture non sono state validate. La compilazione richiede risorse superiori a quelle del bot in esecuzione: eseguirla su una macchina adeguata, senza servizi domestici critici. Il file Compose e la guida sono nuovi; il bot e il Dockerfile sono gia stati provati, ma questa nuova installazione completa deve ancora essere collaudata su un host pulito.

```sh
git clone https://github.com/mrjacock/govd.git
cd govd
curl --fail --location --retry 2 --max-time 90 \
  --output yt-dlp_musllinux.zip \
  https://github.com/yt-dlp/yt-dlp/releases/download/2026.10.26/yt-dlp_musllinux.zip
echo 'c3a67f7c14d4cbf876e7801039237fc16b929a629c72f9c785ec3fdc8aa40748  yt-dlp_musllinux.zip' | sha256sum --check -
```

Se il controllo fallisce, fermarsi. Conservare lo SHA del sorgente installato con `git rev-parse HEAD`.

```sh
umask 077
cp deploy/review.env.example .env.review
mkdir -p private
openssl rand -hex 32
```

Aprire `.env.review` con un editor locale e compilare:

- `BOT_TOKEN`: token del proprio bot ottenuto da BotFather. Non inserirlo in issue o chat pubbliche.
- `ADMINS`: il proprio ID numerico Telegram, non il nome utente. Per piu amministratori, separarli con virgole.
- `WHITELIST`: per iniziare, lo stesso ID personale; autorizza soltanto la chat privata. Non lasciarlo vuoto: nel codice originale una whitelist vuota non limita l'accesso. Compose richiede un valore.
- `DB_PASSWORD`: la stringa casuale appena generata, senza spazi.

Non copiare `private/config-example.yaml`: contiene proxy dimostrativi. Per i casi pubblici Instagram, Facebook e TikTok verificati non servono cookie personali ne un account social. Non vengono importate sessioni del browser. Alcuni estrattori legacy richiedono configurazioni aggiuntive descritte nel README; non promettiamo funzionamento di tutte le piattaforme.

```sh
docker compose --env-file .env.review -f compose.review.yaml config --quiet
docker compose --env-file .env.review -f compose.review.yaml build bot
docker compose --env-file .env.review -f compose.review.yaml up -d
docker compose --env-file .env.review -f compose.review.yaml ps
```

Il database deve risultare healthy e il bot running. Inviare `/start` al proprio bot e poi un collegamento pubblico. Non avviare due istanze con lo stesso token: interferirebbero con la ricezione Telegram. Per leggere gli errori usare `docker compose --env-file .env.review -f compose.review.yaml logs --tail=50 bot`; non pubblicare l'output senza rimuovere token, ID e URL personali. Non condividere l'output completo di `compose config`, che espande i segreti.

## Limiti e accesso ai gruppi

Il file di avvio imposta 50 MiB per file, durata massima 5 minuti, due aggiornamenti simultanei, cache disabilitata e 256 MiB complessivi di filesystem temporanei in RAM. Il contenitore del bot usa al massimo 768 MiB, un core e 128 processi; i log Docker sono ruotati. Il database e gli strati delle immagini occupano spazio persistente e non rientrano nei 256 MiB. Questa configurazione non quota l'intero disco ne la RAM del database.

Per autorizzare un gruppo aggiungere il suo esatto ID numerico (normalmente negativo) a `WHITELIST`, mantenendo anche il proprio. `ADMINS` contiene ID di persone, non ID del gruppo. Tutti i membri del gruppo autorizzato possono richiedere download; i comandi amministrativi del bot restano riservati agli amministratori configurati. Telegram deve consegnare i messaggi al bot: verificare in BotFather la privacy per i gruppi e i permessi effettivi. Non aprire l'accesso generale cancellando la whitelist. Dopo una modifica all'ambiente, ripetere `up -d`.

Il bot limita frequenza, concorrenza e download; questo non equivale a una garanzia contro ogni abuso. Non passare cookie del proprio account a un bot pubblico senza valutarne l'esposizione.

## Isolamento della rete

Compose non pubblica porte del bot o del database e non modifica il firewall dell'host. Tuttavia una normale rete Docker consente connessioni in uscita anche alla LAN. Per usare il bot in gruppi non fidati, aggiungere sul proprio host un isolamento verificato che permetta Internet HTTP/HTTPS, DNS verso resolver espliciti e il database, bloccando host, reti private e servizi metadata. Deve coprire anche IPv6, se abilitato, e il processo yt-dlp. Le protezioni Go sugli URL non coprono tutte le richieste del motore esterno. Vedere REVIEW-IT.md; non sono incluse regole universali da applicare alla cieca.

## Aggiornare un'installazione esistente

Non puntare automaticamente questa procedura al volume del vecchio database: PostgreSQL di versioni principali diverse non puo riutilizzare direttamente la stessa directory. Annotare Compose, immagini, versione PostgreSQL, volume effettivo e sorgente precedenti. Salvare privatamente `.env`, `private/` e un dump coerente del database con gli strumenti della propria versione. Verificare il ripristino su un database di prova prima del passaggio. Conservare la vecchia immagine; `git pull` non costituisce un backup.

Per una prima prova usare un token diverso e il database separato creato da questo Compose. Per il passaggio reale fermare soltanto il vecchio bot, conservare la versione PostgreSQL esistente oppure eseguire una migrazione logica verificata, e avviare una sola istanza con il token reale. Le migrazioni del bot possono partire all'avvio: tornare all'immagine precedente potrebbe richiedere anche il ripristino del database. Non usare `down -v` e non cancellare il vecchio volume.

Per gli aggiornamenti successivi della stessa installazione: backup, `git pull --ff-only`, verifica del sorgente, controllo del checksum dell'helper, quindi `build bot` e `up -d`. Prima di applicare cambiamenti leggere le note del commit. La guida non avvia aggiornamenti automatici.

## Arresto e riavvio

```sh
docker compose --env-file .env.review -f compose.review.yaml stop bot
docker compose --env-file .env.review -f compose.review.yaml start bot
```

`unless-stopped` riavvia il servizio dopo un crash o un riavvio del sistema, ma conserva un arresto manuale. Nessuna di queste istruzioni richiede di riavviare il server. I limiti e le prove del codice sono documentati in VERIFICATION.md.
