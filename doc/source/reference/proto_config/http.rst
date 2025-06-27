.. _proto-config-http:

Configuration HTTP & HTTPS
##########################

Configuration serveur
=====================

* **minTLSVersion** (*string*) - **[HTTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le serveur. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration client
====================

* **minTLSVersion** (*string*) - **[HTTPS uniquement]** Spécifie la version minimale
  de TLS autorisée par le client. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration partenaire
========================

* **minTLSVersion** (*string*) - **[HTTPS uniquement]** Spécifie la version minimale
  de TLS autorisée pour ce partenaire. Les valeurs acceptées sont "v1.0", "v1.1",
  "v1.2" et "v1.3". Par défaut, la version minimale est "v1.2".