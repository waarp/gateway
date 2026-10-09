Configuration Webdav
####################

Configuration serveur
=====================

* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée par le serveur. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".
* **cipherSuites** (*array of string*) - **[TLS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées par le serveur. Si la liste est vide, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.

Configuration client
====================

* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée par le client. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".
* **cipherSuites** (*array of string*) - **[TLS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées par le client. Si la liste est vide, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.

Configuration partenaire
========================

* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée pour ce partenaire. Les valeurs acceptées sont "v1.0", "v1.1",
  "v1.2" et "v1.3". Par défaut, la version minimale est "v1.2".
* **cipherSuites** (*array of string*) - **[TLS uniquement]** Spécifie la liste
  des suites de chiffrement TLS autorisées pour ce partenaire. Cette liste remplace
  celle définie dans la configuration du client. Si les deux listes sont vides, les
  suites de chiffrement par défaut sont utilisées. Voir :ref:`la liste des suites de
  chiffrement acceptées <proto-config-cipher-suites>`.