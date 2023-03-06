.. _reference-backup-json:

#######################
Fichier d'import/export
#######################

Lors de l'import/export de la base de données de la *gateway*, les données sont
stockées dans un fichier en format JSON. Ce JSON a la forme suivante :


* **local** (*array*) - La liste des :term:`serveurs locaux<serveur>` de la
  Gateway.

  * **name** (*string*) - Le nom du serveur.
  * **protocol** (*string*) - Le protocole du serveur.
  * **disabled** (*bool*) - Indique si le serveur doit être démarré automatiquement
    au lancement de la gateway.
  * **root** (*string*) - Le dossier racine du serveur.
  * **workDir** (*string*) - Le dossier temporaire du serveur.
  * **configuration** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * **certificates** (*array*) - La liste des :term:`certificats
    <certificat>` du serveur.

    * **name** (*string*) - Le nom du certificat.
    * **privateKey** (*string*) - La clé privée du serveur en format PEM.
    * **certificat** (*string*) - La chaîne de certification du serveur en
      format PEM.

  * **accounts** (*array*) - La liste des comptes rattaché au serveur.

    * **login** (*string*) - Le login du compte.
    * **password** (*string*) - Le mot de passe du compte. Si le compte a été
      exporté depuis une gateway existante, `passwordHash` sera utilisé à la
      place.
    * **passwordHash** (*string*) - Un hash bcrypt du mot de passe du compte.
      utilisé uniquement lors de l'export depuis une gateway existante.
    * **certificates** (*array*) - La liste des :term:`certificats<certificat>`
      du compte.

      * **name** (*string*) - Le nom du certificat.
      * **Certificat** (*string*) - La chaîne de certification du compte en
        format PEM (mutuellement exclusif avec `public_key`).
      * **publicKey** (*string*) - La clé publique SSH du compte (*hostkey*) en
        format *authorized_key* (mutuellement exclusif avec `certificate`)

* **remotes** (*array*) - La liste des :term:`partenaires<partenaire>` de
  transfert de la gateway.

  * **name** (*string*) - Le nom du partenaire.
  * **protocol** (*string*) - Le protocole du partenaire.
  * **configuration** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * **certificates** (*array*) - La liste des :term:`certificats
    <certificat>` du partenaire.

    * **name** (*string*) - Le nom du certificat.
    * **Certificat** (*string*) - La chaîne de certification du partenaire en
      format PEM (mutuellement exclusif avec `public_key`).
    * **publicKey** (*string*) - La clé publique SSH du partenaire (*hostkey*) en
      format *authorized_key* (mutuellement exclusif avec `certificate`)

  * **accounts** (*array*) - La liste des comptes rattaché au partenaire.

    * **login** (*string*) - Le login du compte.
    * **password** (*string*) - Le mot de passe du compte.
    * **certificates** (*array*) - La liste des :term:`certificats<certificat>`
      du compte.

      * **name** (*string*) - Le nom du certificat.
      * **privateKey** (*string*) - La clé privée du compte en format PEM.
      * **certificat** (*string*) - La chaîne de certification du compte en
        format PEM.

* **clients** (*array*) - La liste des :term:`clients<clients>` de transfert de
  la gateway.

  * **name** (*string*) - Le nom du client.
  * **protocol** (*string*) - Le protocole du client.
  * **disabled** (*bool*) - Indique si le client doit être démarré automatiquement
    au lancement de la gateway.
  * **localAddress** (*string*) - L'adresse locale du client.
  * **protoConfig** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du client.

* **rules** (*array*) - La liste des règles de transfert de la gateway.

  * **name** (*string*) - Le nom de la règle de transfert.
  * **isSend** (*bool*) - Le sens de la règle. ``true`` pour l'envoi, ``false``
    pour la réception.
  * **path** (*string*) - Le chemin de la règle. Permet d'identifier la règle
    lorsque le protocole seul ne le permet pas.
  * **inPath** (*string*) - Le dossier de réception de la règle.
  * **outPath** (*string*) - Le dossier d'envoi de la règle.
  * **workPath** (*string*) - Le dossier de réception temporaire de la règle.
  * **auth** (*array*) - La liste des agents autorisés à utiliser la règles.
    Chaque élément de la liste doit être précédé de sa nature (``remote`` ou
    ``local``) suivi du nom de l'agent, le tout séparé par ``::`` (ex:
    ``local::serveur_sftp``). Si l'agent est un compte, alors le nom de compte
    doit être précédé du nom du serveur/partenaire auquel le compte est
    rattaché (ex: ``local::serveur_sftp::toto``).
  * **pre** (*array*) - La liste des pré-traitements de la règle. Voir la
    :any:`documentation <reference-tasks>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun d'entre
    eux.

    * **type** (*string*) - Le type de traitement.
    * **args** (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

  * **post** (*array*) - La liste des post-traitements de la règle. Voir la
    :any:`documentation <reference-tasks>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun
    d'entre eux.

    * **type** (*string*) - Le type de traitement.
    * **args** (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

  * **error** (*array*) - La liste des traitements d'erreur de la règle. Voir
    la :any:`documentation<tasks/index>` des traitements pour la liste des
    traitements disponibles ainsi que les arguments nécessaires à chacun
    d'entre eux.

    * **type** (*string*) - Le type de traitement.
    * **args** (*object*) - Les arguments du traitement. Variable suivant le
      type de traitement (cf. :any:`traitements <reference-tasks>`).

.. note:: Les éléments suivants du fichier d'import sont dépréciés, et seront
   prochainement retirés. En conséquence, ils ne devraient donc plus être
   utilisés si possible.

* **remotes** (*array*) - La liste des :term:`partenaires<partenaire>` de
  transfert de la gateway. [**OBSOLÈTE**] Remplacé par ``clients.partners``.

  * **name** (*string*) - Le nom du partenaire.
  * **protocol** (*string*) - Le protocole du partenaire.
  * **configuration** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * **certificates** (*array*) - La liste des :term:`certificats
    <certificat>` du partenaire.

    * **name** (*string*) - Le nom du certificat.
    * **Certificat** (*string*) - La chaîne de certification du partenaire en
      format PEM (mutuellement exclusif avec `public_key`).
    * **publicKey** (*string*) - La clé publique SSH du partenaire (*hostkey*) en
      format *authorized_key* (mutuellement exclusif avec `certificate`)

  * **accounts** (*array*) - La liste des comptes rattaché au partenaire.

    * **login** (*string*) - Le login du compte.
    * **password** (*string*) - Le mot de passe du compte.
    * **certificates** (*array*) - La liste des :term:`certificats<certificat>`
      du compte.

      * **name** (*string*) - Le nom du certificat.
      * **privateKey** (*string*) - La clé privée du compte en format PEM.
      * **certificat** (*string*) - La chaîne de certification du compte en
        format PEM.


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
     "clients": [{
       "name": "sftp_client",
       "protocol": "sftp",
       "disabled": false,
       "localAddress": "0.0.0.0:2223",
       "partners": [{
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
     }]
   }
