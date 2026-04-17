Configuration AS2
#################

Configuration serveur
=====================

* **maxFileSize** (*number*) - Spécifie la taille maximale (en octets) autorisée
  pour les fichiers sur ce serveur. Les fichiers de taille supérieure seront
  refusés. Ne peut être supérieur à la quantité de mémoire totale du système.
  Le défaut est de 1Mo (1 000 000 d'octets).
* **mdnSignatureAlgorithm** (*string*) - L'algorithme à utiliser pour signer
  les MDNs d'acquittement envoyés par le serveur. Nécessite obligatoirement qu'un
  certificat x509 soit attaché au serveur. Laisser vide pour désactiver la
  signature de MDN. Valeurs acceptées :

  - "sha1"
  - "md5"
  - "sha256"
  - "sha384"
  - "sha512"
* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée par le serveur. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration client
====================

* **maxFileSize** (*number*) - Spécifie la taille maximale (en octets) autorisée
  pour les fichiers sur ce client. Les fichiers de taille supérieure seront
  refusés. Ne peut être supérieur à la quantité de mémoire totale du système.
  Le défaut est de 1Mo (1 000 000 d'octets).
* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée par le client. Les valeurs acceptées sont "v1.0", "v1.1", "v1.2"
  et "v1.3". Par défaut, la version minimale est "v1.2".

Configuration partenaire
========================

* **signatureAlgorithm** (*string*) - L'algorithme à utiliser pour signer
  les requêtes envoyées à ce partenaire. *Nécessite obligatoirement qu'un
  certificat x509 soit attaché au compte distant utilisé pour le transfert*.
  Laisser vide pour désactiver la signature de requête. Valeurs acceptées :

  - "sha1"
  - "md5"
  - "sha256"
  - "sha384"
  - "sha512"
* **encryptionAlgorithm** (*string*) - L'algorithme à utiliser pour chiffrer
  les requêtes envoyées à ce partenaire. *Nécessite obligatoirement qu'un
  certificat x509 soit attaché au partenaire en question*.
  Laisser vide pour désactiver le chiffrement des requêtes. Valeurs acceptées :

  - "des-cbc"
  - "aes128-cbc"
  - "aes128-gcm"
  - "aes256-cbc"
  - "aes256-gcm"
* **asyncMDNAddress** (*string*) - L'adresse à laquelle les MDNs d'acquittement
  asynchrones du partenaire doivent être envoyés. Laisser vide pour utiliser des
  acquittements synchrones.
* **handleAsyncMDN** (*boolean*) - Indique si Gateway doit gérer directement le
  MDN d'acquittement asynchrone. Cette option n'a pas d'effet si les MDNs sont
  synchrones.

  Si activée, Gateway écoutera sur l'adresse indiquée dans *asyncMDNAddress* et
  attendra un acquittement pour terminer le transfert. Pour cette raison, cette
  option ne peut être activée que si *asyncMDNAddress* a été renseignée.

  Si désactivée, le MDN asynchrone est considéré comme géré par une application
  tierce, et Gateway n'attendra pas d'acquittement pour terminer le transfert.
* **minTLSVersion** (*string*) - **[TLS uniquement]** Spécifie la version minimale
  de TLS autorisée pour ce partenaire. Les valeurs acceptées sont "v1.0", "v1.1",
  "v1.2" et "v1.3". Par défaut, la version minimale est "v1.2".