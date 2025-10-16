.. _reference-backup-json:

#######################
Fichier d'import/export
#######################

Lors de l'import/export de la base de données de Gateway, les données sont
stockées dans un fichier en format JSON ou YAML. Ce fichier a la forme suivante :


* ``local`` (*array*) - La liste des :term:`serveurs locaux<serveur>` de la
  Gateway.

  * ``name`` (*string*) - Le nom du serveur.
  * ``protocol`` (*string*) - Le protocole du serveur.
  * ``disabled`` (*bool*) - Indique si le serveur doit être démarré automatiquement
    au lancement de la gateway.
  * ``root`` (*string*) - Le dossier racine du serveur.
  * ``workDir`` (*string*) - Le dossier temporaire du serveur.
  * ``configuration`` (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * ``credentials`` (*array*) - La liste des :term:`informations d'authentification
    <information d'authentification>` du serveur. Voir la page sur les
    :ref:`méthodes d'authentification <reference-auth-methods>` (section
    "authentification externe") pour la liste des types supportés et les valeurs
    d'authentification attendue pour chacun d'entres eux.

    * ``name`` (*string*) - [Optionnel] Le nom de l'identifiant. Par défaut,
      le type est utilisé comme nom.
    * ``type`` (*string*) - Le type d'identifiant.
    * ``value`` (*string*) - La valeur d'authentification (mot de passe,
      certificat, ...).
    * ``value2`` (*string*) - [Optionnel] La valeur d'authentification
      secondaire (si la méthode d'authentification en requiert une).
  * ``certificates`` (*array*) - La liste des :term:`certificats
    <certificat>` du serveur. [**OBSOLÈTE**] Remplacé par ``credentials``.

    * ``name`` (*string*) - Le nom du certificat.
    * ``privateKey`` (*string*) - La clé privée du serveur en format PEM.
    * ``Certificate`` (*string*) - La chaîne de certification du serveur en
      format PEM.

  * ``accounts`` (*array*) - La liste des comptes rattaché au serveur.

    * ``login`` (*string*) - Le login du compte.
    * ``password`` (*string*) - Le mot de passe du compte. Si le compte a été
      exporté depuis une gateway existante, ``passwordHash`` sera utilisé à la
      place.
    * ``passwordHash`` (*string*) - Un hash bcrypt du mot de passe du compte.
      utilisé uniquement lors de l'export depuis une gateway existante.
    * ``credentials`` (*array*) - La liste des :term:`informations d'authentification
      <information d'authentification>` du compte. Voir la page sur les
      :ref:`méthodes d'authentification <reference-auth-methods>` (section
      "authentification interne") pour la liste des types supportés et les
      valeurs d'authentification attendue pour chacun d'entres eux.

      * ``name`` (*string*) - [Optionnel] Le nom de l'identifiant. Par défaut,
        le type est utilisé comme nom.
      * ``type`` (*string*) - Le type d'identifiant.
      * ``value`` (*string*) - La valeur d'authentification (mot de passe,
        certificat, ...).
      * ``value2`` (*string*) - [Optionnel] La valeur d'authentification
        secondaire (si la méthode d'authentification en requiert une).
    * ``certificates`` (*array*) - La liste des :term:`certificats<certificat>`
      du compte. [**OBSOLÈTE**] Remplacé par ``credentials``.

      * ``name`` (*string*) - Le nom du certificat.
      * ``Certificate`` (*string*) - La chaîne de certification du compte en
        format PEM (mutuellement exclusif avec ``public_key``).
      * ``publicKey`` (*string*) - La clé publique SSH du compte (*hostkey*) en
        format *authorized_key* (mutuellement exclusif avec ``certificate``)

* ``remotes`` (*array*) - La liste des :term:`partenaires<partenaire>` de
  transfert de Gateway.

  * ``name`` (*string*) - Le nom du partenaire.
  * ``protocol`` (*string*) - Le protocole du partenaire.
  * ``configuration`` (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * ``credentials`` (*array*) - La liste des :term:`informations d'authentification
    <information d'authentification>` du partenaire. Voir la page sur les
    :ref:`méthodes d'authentification <reference-auth-methods>` (section
    "authentification interne") pour la liste des types supportés et les valeurs
    d'authentification attendue pour chacun d'entres eux.

    * ``name`` (*string*) - [Optionnel] Le nom de l'identifiant. Par défaut,
      le type est utilisé comme nom.
    * ``type`` (*string*) - Le type d'identifiant.
    * ``value`` (*string*) - La valeur d'authentification (mot de passe,
      certificat, ...).
    * ``value2`` (*string*) - [Optionnel] La valeur d'authentification
      secondaire (si la méthode d'authentification en requiert une).
  * ``certificates`` (*array*) - La liste des :term:`certificats
    <certificat>` du partenaire. [**OBSOLÈTE**] Remplacé par ``credentials``.

    * ``name`` (*string*) - Le nom du certificat.
    * ``Certificate`` (*string*) - La chaîne de certification du partenaire en
      format PEM (mutuellement exclusif avec ``public_key``).
    * ``publicKey`` (*string*) - La clé publique SSH du partenaire (*hostkey*) en
      format *authorized_key* (mutuellement exclusif avec ``certificate``)

  * ``accounts`` (*array*) - La liste des comptes rattaché au partenaire.

    * ``login`` (*string*) - Le login du compte.
    * ``password`` (*string*) - Le mot de passe du compte.
    * ``credentials`` (*array*) - La liste des :term:`informations d'authentification
      <information d'authentification>` du compte. Voir la page sur les
      :ref:`méthodes d'authentification <reference-auth-methods>` (section
      "authentification externe") pour la liste des types supportés et les
      valeurs d'authentification attendue pour chacun d'entres eux.

      * ``name`` (*string*) - [Optionnel] Le nom de l'identifiant. Par défaut,
        le type est utilisé comme nom.
      * ``type`` (*string*) - Le type d'identifiant.
      * ``value`` (*string*) - La valeur d'authentification (mot de passe,
        certificat, ...).
      * ``value2`` (*string*) - [Optionnel] La valeur d'authentification
        secondaire (si la méthode d'authentification en requiert une).
    * ``certificates`` (*array*) - La liste des :term:`certificats<certificat>`
      du compte. [**OBSOLÈTE**] Remplacé par ``credentials``.

      * ``name`` (*string*) - Le nom du certificat.
      * ``privateKey`` (*string*) - La clé privée du compte en format PEM.
      * ``Certificate`` (*string*) - La chaîne de certification du compte en
        format PEM.

* ``clients`` (*array*) - La liste des :term:`clients<client>` de transfert de
  la gateway.

  * ``name`` (*string*) - Le nom du client.
  * ``protocol`` (*string*) - Le protocole du client.
  * ``disabled`` (*bool*) - Indique si le client doit être démarré automatiquement
    au lancement de la gateway.
  * ``localAddress`` (*string*) - L'adresse locale du client.
  * ``protoConfig`` (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du client.
  * ``nbOfAttempts`` (*number*) - Le nombre de fois qu'un transfert sera automatiquement
    re-tenté en cas d'échec (n'inclue pas la tentative originale du transfert).
  * ``firstRetryDelay`` (*number*) - Le délai (en secondes) entre la tentative originale
    des transferts et leur première reprise automatique.
  * ``retryIncrementFactor`` (*number*) - Le facteur par lequel le délai entre chaque
    tentative de transfert est multiplié après chaque essai. Les nombres décimaux
    (ex: 1.5) sont acceptés.

* ``rules`` (*array*) - La liste des règles de transfert de la gateway.

  * ``name`` (*string*) - Le nom de la règle de transfert.
  * ``isSend`` (*bool*) - Le sens de la règle. ``true`` pour l'envoi, ``false``
    pour la réception.
  * ``path`` (*string*) - Le chemin de la règle. Permet d'identifier la règle
    lorsque le protocole seul ne le permet pas.
  * ``inPath`` (*string*) - Le dossier de réception de la règle.
  * ``outPath`` (*string*) - Le dossier d'envoi de la règle.
  * ``workPath`` (*string*) - Le dossier de réception temporaire de la règle.
  * ``auth`` (*array*) - La liste des agents autorisés à utiliser la règle.
    Chaque élément de la liste doit être précédé de sa nature (``remote`` ou
    ``local``) suivi du nom de l'agent, le tout séparé par ``::`` (ex:
    ``local::serveur_sftp``). Si l'agent est un compte, alors le nom de compte
    doit être précédé du nom du serveur/partenaire auquel le compte est
    rattaché (ex: ``local::serveur_sftp::toto``).
  * ``pre`` (*array*) - La liste des pré-traitements de la règle. Voir la
    :any:`documentation <reference-tasks>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun d'entre
    eux.

    * ``type`` (*string*) - Le type de traitement.
    * ``args`` (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

  * ``post`` (*array*) - La liste des post-traitements de la règle. Voir la
    :any:`documentation <reference-tasks>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun
    d'entre eux.

    * ``type`` (*string*) - Le type de traitement.
    * ``args`` (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

  * ``error`` (*array*) - La liste des traitements d'erreur de la règle. Voir
    la :any:`documentation<tasks/index>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun
    d'entre eux.

    * ``type`` (*string*) - Le type de traitement.
    * ``args`` (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

* ``users`` (*array*) - La liste des utilisateurs de l'interface d'administration
  de la gateway.

  * ``username`` (*string*) - Le nom de l'utilisateur.
  * ``password`` (*string*) - Le mot de passe en clair de l'utilisateur.
    Utilisé uniquement pour l'import, les mots de passes ne sont jamais exportés
    en clair mais sous forme de hash (voir ci-dessous).
  * ``passwordHash`` (*string*) - Un hash bcrypt du mot de passe de l'utilisateur.
  * ``permissions`` (*object*) - La liste des droits de l'utilisateur. Les droits
    sont renseignés en format chmod ("rwd") indiquant respectivement le droit de
    lecture, d'écriture et de suppression sur l'élément concerné. Un trait d'union
    "-" est utilisé pour marquer l'absence d'un droit.

    * ``transfers`` (*string*) - Les droits de l'utilisateur en matière de
      gestion transferts.
    * ``servers`` (*string*) - Les droits de l'utilisateur en matière de gestion
      des serveurs et clients locaux.
    * ``partners`` (*string*) - Les droits de l'utilisateur en matière de
      gestion des partenaires distants.
    * ``rules`` (*string*) - Les droits de l'utilisateur en matière de gestion
      des règles de transfert.
    * ``users`` (*string*) - Les droits de l'utilisateur en matière de gestion
      des utilisateurs de l'interface d'administration.
    * ``administration`` (*string*) - Les droits de l'utilisateur en matière de
      gestion de la configuration de Gateway. Cela inclue l'*override* de
      configuration, la gestion de SNMP, des instances cloud et des
      autorités de certification.

* ``clouds`` (*array*) - La liste des instances cloud de la gateway.

  * ``name`` (*string*) - Le nom de l'instance cloud.
  * ``type`` (*string*) - Le type de l'instance cloud. Voir la :ref:`section
    cloud <reference-cloud>` de la documentation pour la liste des types d'instance
    cloud supportés.
  * ``key`` (*string*) - La clé de connexion à l'instance cloud (si l'instance
    cloud en requiert une).
  * ``secret`` (*string*) - Le secret d'authentification (mot de passe, token...)
    de l'instance cloud (si l'instance cloud en requiert un).
  * ``options`` (*object*) - Les options de connexion à l'instance cloud. Ces
    options varie en fonction du type de l'instance cloud. Voir la :ref:`section
    cloud <reference-cloud>` du type de l'instance pour avoir la liste des
    options disponibles.

* ``snmpConfig`` (*object*) - La configuration SNMP.

  * ``server`` (*object*) - La configuration du serveur SNMP local.

    * ``localUDPAddress`` (*string*) - L'adresse UDP locale (port inclus) du
      serveur SNMP.
    * ``v3Only`` (*bool*) - Indique si le serveur est restreint à SNMPv3 uniquement.
      Par défaut, SNMPv2 et SNMPv3 sont toutes deux acceptées (à supposé que leurs
      configurations respectives ci-dessous soient valides).
    * ``community`` (*string*) - [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du serveur. Par défaut, la valeur "public" est utilisée.
    * ``v3Username`` (*string*) - [SNMPv3 uniquement] Le nom d'utilisateur pour
      l'authentifier sur le serveur. À noter que le nom d'utilisateur est requis
      avec SNMPv3 même si l'authentification est désactivée.
    * ``v3AuthProtocol`` (*string*) - [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
    * ``v3AuthPassphrase`` (*string*) - [SNMPv3 uniquement] La passphrase d'authentification.
    * ``v3PrivacyProtocol`` (*string*) - [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
    * ``v3PrivacyPassphrase`` (*string*) - [SNMPv3 uniquement] La passphrase de confidentialité.

  * ``monitors`` (*array*) - La liste des moniteurs SNMP connus.

    * ``name`` (*string*) - Le nom du moniteur SNMP.
    * ``snmpVersion`` (*string*) - La version de SNMP utilisée par le moniteur.
      Les versions acceptées sont "SNMPv2" et "SNMPv3" (SNMPv1 n'est pas supportée).
    * ``udpAddress`` (*string*) - L'adresse UDP (port inclus) du moniteur à laquelle
      les notifications SNMP doivent être envoyées.
    * ``useInforms`` (*bool*) - Spécifie le type de notification à envoyer au moniteur.
      Si *faux* (par défaut), Gateway enverra des *traps*. Si *vrai*, Gateway
      enverra des *informs*.
    * ``community`` (*string*) - [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du moniteur. Par défaut, la valeur "public" est utilisée.
    * ``v3ContextName`` (*string*) - [SNMPv3 uniquement] Le nom du contexte SNMPv3.
    * ``v3ContextEngineID`` (*string*) - [SNMPv3 uniquement] L'ID du moteur de contexte SNMPv3.
    * ``v3Security`` (*string*) - [SNMPv3 uniquement] Spécifie le niveau de
      sécurité SNMPv3 à utiliser avec ce moniteur. Les valeurs acceptées sont :

         - ``noAuthNoPriv``: pas d'authentification ni de confidentialité
         - ``authNoPriv``: authentification, mais pas de confidentialité
         - ``authPriv``: authentification et confidentialité

      Par défaut, l'authentification et la confidentialité sont toutes deux
      désactivées.
    * ``authEngineID`` (*string*) - [SNMPv3 uniquement] L'ID du moteur d'authentification.
      N'a aucun effet si le moniteur utilise des *informs* (voir l'option *useInforms*
      ci-dessus).
    * ``authUsername`` (*string*) - [SNMPv3 uniquement] Le nom d'utilisateur. À noter
      que le nom d'utilisateur est requis avec SNMPv3 même si l'authentification
      est désactivée.
    * ``authProtocol`` (*string*) - [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
    * ``authPassphrase`` (*string*) - [SNMPv3 uniquement] La passphrase d'authentification.
    * ``privProtocol`` (*string*) - [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
    * ``privPassphrase`` (*string*) - [SNMPv3 uniquement] La passphrase de confidentialité.

* ``authorities`` (*array*) - Liste des autorités de certification reconnue par Gateway.

  * ``name`` (*string*) - Le nom de l'autorité.
  * ``type`` (*string*) - Le type d'autorité (voir les :ref:`méthodes d'authentification
    <reference-auth-methods>`, chapitre "Autorité d'authentification" pour la liste
    des types supportés.
  * ``publicIdentity`` (*string*) - La valeur d'identité publique de l'autorité
    (en général, son certificat).
  * ``validHosts`` (*array of strings*) - La liste des hôtes que l'autorité est
    habilitée à authentifier. Si vide, l'autorité est habilité à authentifier tous
    les hôtes qu'elle a certifié.

* ``cryptoKeys`` (*array*) - Liste des clés cryptographiques de la Gateway.

  * ``name`` (*string*) - Le nom de la clé
  * ``type`` (*string*) - Le type de la clé. Les valeurs autorisées sont :

       - ``AES`` pour une clé de chiffrement AES
       - ``HMAC`` pour une clé de signature HMAC
       - ``PGP-PUBLIC`` pour les clés PGP publiques
       - ``PGP-PRIVATE`` pour les clés PGP privées
  * ``key`` (*string*) - La clé en format textuel. Si la clé n'a pas de format
    textuel natif, la clé doit être fournie en format Base64.

**Exemple**

.. code-block:: json

   {
     "locals": [{
       "name": "serveur_sftp",
       "protocol": "sftp",
       "disabled": false,
       "address": "0.0.0.0:2222"
       "root": "/sftp",
       "workDir": "/sftp/tmp",
       "accounts": [{
         "login": "toto",
         "password": "sésame",
         "certs": [{
           "name": "toto_ssh_pbk",
           "publicKey": "<clé publique SSH>",
         }]
       }],
       "certs": [{
         "name": "server_sftp_hostkey",
         "privateKey": "<clé privée SSH>",
       }]
     }],
     "remotes": [{
       "name": "openssh",
       "address": "10.0.0.0:22"
       "accounts": [{
         "login": "titi",
         "password": "sésame",
         "certs": [{
           "name": "titi_ssh_pk",
           "privateKey": "<clé privée SSH>",
         }]
       }],
       "certs": [{
         "name": "openssh_hostkey",
         "publicKey": "<clé publique SSH>",
       }]
     }]
     "clients": [{
       "name": "sftp_client",
       "protocol": "sftp",
       "disabled": false,
       "localAddress": "0.0.0.0:2223",
     }],
     "rules": [{
       "name": "send",
       "isSend": true,
       "path": "send",
       "inPath": "send/in",
       "outPath": "send/out",
       "workPath": "send/tmp",
       "access": [
         "local::serveur_sftp",
         "remote::openssh"
       ],
       "pre": [],
       "post": [],
       "error": []
     }, {
       "name": "receive",
       "isSend": false,
       "path": "receive",
       "access": [
         "local::openssh",
         "local::serveur_sftp::toto",
       ],
       "pre": [],
       "post": [],
       "error": []
     }],
     "users": [{
       "username": "toto",
       "password": "sésame",
       "permissions": {
         "transfers": "rw-",
         "servers": "rwd",
         "partners": "rw-",
         "rules": "rwd",
         "users": "r--",
         "administration": "---"
       }
     }],
     "clouds": [{
       "name": "aws-s3",
       "type": "s3",
       "key": "<clé d'accès AWS>",
       "secret": "<clé d'accès secrète AWS>",
       "options": {
         "region": "eu-west-1",
         "bucket": "gw-bucket",
       }
     }],
     "snmpConfig": {
       "server" : {
         "localUDPAddress": "0.0.0.0:161",
         "community": "public",
         "v3Only": false,
         "v3Username": "toto"
         "v3AuthProtocol": "MD5"
         "v3AuthPassphrase": "sesame",
         "v3PrivacyProtocol": "DES",
         "v3PrivacyPassphrase": "secret"
       },
       "monitors" : [{
         "name": "centreon",
         "snmpVersion": "SNMPv3",
         "udpAddress": "20.0.0.0:162"
         "community": "public",
         "useInforms": true,
         "v3ContextName": "waarp-gw",
         "v3ContextEngineID": "123"
         "v3Security": "authPriv",
         "v3AuthEngineID": "456",
         "v3AuthUsername": "tata",
         "v3AuthProtocol": "SHA",
         "v3AuthPassphrase": "sesame",
         "v3PrivacyProtocol": "AES",
         "v3PrivacyPassphrase": "secret"
       }]
     },
     "authorities": [{
       "name": "cert_authority",
       "type": "tls_authority",
       "publicIdentity": "<certificat de l'autorité en format PEM>"
       "validHosts": ["192.168.1.1", "waarp.fr"]
     }],
     "cryptoKeys": [{
       "name": "aes-key",
       "type": "AES",
       "key": "<clé AES en Base64>"
     }, {
       "name": "pgp-key",
       "type": "PGP-PRIVATE",
       "key": "<clé PGP en format PEM>"
     }]
   }
