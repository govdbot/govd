# Govd: versione da revisionare — 31 ottobre 2026

Base upstream: c49232db414f2a1132c505b0e5833813beaed3ea (marzo 2026).
Questa copia non contiene token Telegram, password, ID personali, cookie o sessioni Instagram. La licenza MIT originale resta inclusa. Pubblicata come copia indipendente per revisione, senza approvazione upstream.

## Cosa cambia

- Fallback Instagram anonimo opzionale con yt-dlp: Reel, post fotografici, album misti e link Condividi. Senza yt-dlp rimangono i metodi nativi. Non promette accesso a contenuti privati, cancellati o limitati da Instagram.
- Facebook pubblico tramite il motore locale già usato per Instagram; TikTok tramite i dati mobile/embed e i cookie anonimi temporanei emessi dal sito. Non vengono importati account o sessioni personali. Le richieste del file mantengono le stesse intestazioni e il download può recuperare un rifiuto dei blocchi con un singolo trasferimento normale, sempre entro i limiti.
- Identificatori completi dei lavori, controllo del proprietario basato sull'identità Telegram e consumo atomico; autorizzazione degli eventi inline e statistiche riservate all'amministratore.
- Controlli degli indirizzi, dei reindirizzamenti e dei cookie; connessione agli stessi IP pubblici verificati dopo la risoluzione DNS. Il motore esterno richiede anche isolamento di rete del sistema per la medesima protezione.
- Limiti di frequenza e concorrenza, cache e code limitate; gestione di URL in testo, didascalie e collegamenti incorporati, fino a tre link unici per messaggio.
- Limite effettivo dei byte scaricati e dei file finali, timeout di estrazione/FFmpeg, chiusura degli upload e pulizia dei temporanei. Miniature opzionali che non fanno fallire il video.
- Go 1.26.10 e aggiornamento delle dipendenze interessate dalla scansione.

## Installazione e aggiornamento

Non applicare la patch alla cieca a una versione diversa dalla base indicata. Eseguire prima un backup di sorgenti, configurazione e database; provare la nuova immagine con il proprio bot di test. La patch non modifica credenziali o autorizzazioni e non apre automaticamente il bot ai gruppi.

Per abilitare il fallback installare yt-dlp compatibile (testato 2026.10.26, con impersonazione Chrome). Dockerfile.group documenta l'immagine verificata: richiede nel contesto l'archivio ufficiale yt-dlp_musllinux.zip, SHA256 c3a67f7c14d4cbf876e7801039237fc16b929a629c72f9c785ec3fdc8aa40748. Non eseguire installer ricevuti in chat o scaricare binari senza verificarne origine e checksum. Il Dockerfile non contiene né installa regole firewall.

La configurazione del bot conserva WHITELIST/ADMINS e gli estrattori abilitati. Per un gruppo, il proprietario deve autorizzare esplicitamente il relativo ID e verificare le impostazioni Telegram di ricezione dei messaggi. Il bot non legge messaggi che Telegram non gli consegna. Non sono richiesti account o cookie Instagram per i casi pubblici verificati.

Esempio di limiti di distribuzione verificati, da adattare esplicitamente: MAX_FILE_SIZE=52428800, MAX_DURATION=5m, CONCURRENT_UPDATES=2, CACHING=false. Contenitore senza root, filesystem in sola lettura, cap_drop ALL, no-new-privileges, 768 MiB RAM senza swap, un core, 128 processi, tmpfs download 224 MiB + /tmp 32 MiB, log ruotati 5 MiB x2. Due lavori contemporanei globali, uno per utente; tre richieste/minuto per utente (sei per amministratore in privato), sei per chat e dodici globali. Le richieste oltre limite ricevono un messaggio di attesa e non entrano in una coda illimitata.

## Isolamento di rete facoltativo ma raccomandato

Le regole firewall di una singola installazione non fanno parte della patch. Nessuno script applica automaticamente modifiche al firewall del sistema. Per una propria installazione consentire soltanto HTTP/HTTPS pubblici, DNS verso resolver espliciti e il database del bot; negare host, LAN, metadata e altre reti private. Adattare subnet, resolver e persistenza alla propria piattaforma e verificare che la rete non cambi dopo un aggiornamento. Usare gli indirizzi della propria installazione.

Questa protezione comprende il processo esterno yt-dlp, oltre ai controlli nel codice Go. Senza isolamento del sistema, il controllo Go non copre tutte le richieste originate dal motore esterno. Nessun account Instagram personale è necessario o incluso.

## Verifiche e limiti

Suite completa Go e test di concorrenza sui componenti modificati superati. Matrice reale: quattro Reel (compresi errori precedenti), album di tre foto e link Condividi scaricati; un video oltre 50 MiB correttamente rifiutato. La matrice verifica anche la rimozione dei temporanei. I test live sono opzionali tramite GOVD_LIVE_MATRIX e non inviano messaggi Telegram.

Govulncheck con toolchain aggiornata: zero vulnerabilità nei simboli raggiungibili e nei pacchetti importati; rimangono due avvisi solo a livello del modulo AWS transitivo, per funzioni S3 Crypto non usate. Non equivale a una certificazione di sicurezza completa né a una scansione esaustiva dei componenti nativi/Python. Gli estrattori di tutte le altre piattaforme non sono stati verificati con una matrice live equivalente.

Prima di pubblicare, separare la proposta funzionale dalle correzioni di sicurezza e coordinare privatamente i dettagli sensibili con il manutentore. Un fork appartiene al proprio account; una pull request propone le modifiche al progetto originale e viene accettata o rifiutata dal manutentore. La patch può essere distribuita separatamente anche senza accettazione upstream.

## Verifica Facebook/TikTok del 31 ottobre

Suite completa e controlli di concorrenza eseguiti su GitHub; 13 casi di download reali riusciti fra Facebook, TikTok e regressioni Instagram. Sei video Facebook distinti e due TikTok distinti provati, con link da app e varianti mobile. Verificati il limite di 50 MiB, la pulizia dei file e la consegna Telegram di due link nello stesso messaggio. Due ulteriori collegamenti Facebook non sono recuperabili nella sessione anonima del bot: non sono conteggiati come video scaricati. La riproduzione da un browser con account non prova la disponibilità anonima. Dettagli in VERIFICATION.md.
