.. _proto-config-ftp:

Configuration FTP et FTPS
#########################

Configuration serveur
=====================

* **disablePassiveMode** (*boolean*) - Désactive le mode FTP passif. Par défaut,
  le mode passif est activé.
* **disableActiveMode** (*boolean*) - Désactive le mode FTP actif. Par défaut,
  le mode actif est activé.
* **passiveModeMinPort** (*integer*) - N° de port minimal de la plage de ports
  utilisés en mode FTP passif (si le mode passif est activé). Par défaut,
  le port minimal est 10000.
* **passiveModeMaxPort** (*integer*) - N° de port maximal de la plage de ports
  utilisés en mode FTP passif (si le mode passif est activé). Par défaut,
  le port maximal est 20000.

* **tlsRequirement** (*string*) - **[FTPS uniquement]** Spécifie le mode TLS
  utilisé par le serveur. Les valeurs acceptées sont "Optional" (TLS explicite
  optionnel), "Mandatory" (TLS explicite obligatoire) et "Implicit" (TLS implicite).
  Par défaut, le mode TLS est "Optional". Voir la section TLS dans la
  :ref:`présentation FTP <ref-proto-ftp>`
* **minTLSVersion** (*string*) - **[FTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le serveur. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration client
====================

* **enablePassiveMode** (*boolean*) - Active le mode FTP actif. Par défaut,
  le mode actif est désactivé.
* **activeModeAddress** (*string*) - Adresse IP locale du client en mode
  actif (si le mode actif est activé). Par défaut, l'adresse IP est 0.0.0.0.
* **activeModeMinPort** (*integer*) - N° de port minimal de la plage de ports
  utilisés en mode FTP actif (si le mode actif est activé). Par défaut,
  le port minimal est 10000.
* **activeModeMaxPort** (*integer*) - N° de port maximal de la plage de ports
  utilisés en mode FTP actif (si le mode actif est activé). Par défaut,
  le port maximal est 20000.

* **minTLSVersion** (*string*) - **[FTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le client. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration partenaire
========================

* **disableActiveMode** (*boolean*) - Désactive le mode FTP actif pour ce
  partenaire spécifiquement (en supposant que le client utilisé pour le
  transfert autorise le mode actif). Par défaut, le mode actif est activé si
  le client l'autorise.
* **disableEPSV** (*boolean*) - Désactive EPSV (ou Extended Passive Mode) pour
  ce partenaire spécifiquement. Par défaut, EPSV est activé mais certains
  serveurs FTP ne supportent pas cette fonctionnalité.

* **useImplicitTLS** (*boolean*) - **[FTPS uniquement]** Spécifie si le partenaire
  doit utiliser le TLS implicite ou explicite. Par défaut, TLS implicite est utilisé.
* **minTLSVersion** (*string*) - **[FTPS uniquement]** Spécifie la version minimale
  de TLS autorisée pour ce partenaire. Les valeurs acceptées sont "v1.0", "v1.1",
  "v1.2" et "v1.3". Par défaut, la version minimale est "v1.2".
* **disableTLSSessionReuse** (*boolean*) - **[FTPS uniquement]** Désactive la
  réutilisation de session TLS avec ce partenaire. Par défaut, les sessions TLS
  sont réutilisées quand cela est possible pour améliorer les performances.
  Cependant, cela peut causer des problèmes de compatibilité avec certains serveurs
  tiers.
