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
  n'importe quel port libre est autorisé.
* **passiveModeMaxPort** (*integer*) - N° de port maximal de la plage de ports
  utilisés en mode FTP passif (si le mode passif est activé). Par défaut,
  n'importe quel port libre est autorisé.

* **tlsRequirement** (*string*) - **[FTPS uniquement]** Spécifie le mode TLS
  utilisé par le serveur. Les valeurs acceptées sont "Optional" (TLS explicite
  optionnel), "Mandatory" (TLS explicite obligatoire) et "Implicit" (TLS implicite).
  Par défaut, le mode TLS est "Optional". Voir la section TLS dans la
  :ref:`présentation FTP <ref-proto-ftp>`
* **minTLSVersion** (*string*) - **[FTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le serveur. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".
* **cipherSuites** (*array of string*) - **[FTPS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées par le serveur. Si la liste est vide, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.

Configuration client
====================

* **enableActiveMode** (*boolean*) - Active le mode FTP actif. Par défaut,
  le mode actif est désactivé.
* **activeModeAddress** (*string*) - Adresse IP locale du client en mode
  actif (si le mode actif est activé). Par défaut, l'adresse IP est 0.0.0.0.
* **activeModeMinPort** (*integer*) - N° de port minimal de la plage de ports
  utilisés en mode FTP actif (si le mode actif est activé). Par défaut,
  n'importe quel port libre est autorisé.
* **activeModeMaxPort** (*integer*) - N° de port maximal de la plage de ports
  utilisés en mode FTP actif (si le mode actif est activé). Par défaut,
  n'importe quel port libre est autorisé.

* **minTLSVersion** (*string*) - **[FTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le client. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".
* **cipherSuites** (*array of string*) - **[FTPS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées par le client. Si la liste est vide, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.

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
* **cipherSuites** (*array of string*) - **[FTPS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées pour ce partenaire. Cette liste remplace
  celle définie dans la configuration du client. Si les deux listes sont vides, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.
* **disableTLSSessionReuse** (*boolean*) - **[FTPS uniquement]** Désactive la
  réutilisation de session TLS avec ce partenaire. Par défaut, les sessions TLS
  sont réutilisées quand cela est possible pour améliorer les performances.
  Cependant, cela peut causer des problèmes de compatibilité avec certains serveurs
  tiers.
