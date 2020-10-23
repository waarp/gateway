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
  * **root** (*string*) - Le dossier racine du serveur.
  * **workDir** (*string*) - Le dossier temporaire du serveur.
  * **configuration** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * **certificates** (*array*) - La liste des :term:`certificats
    <certificat>` du serveur.

    * **name** (*string*) - Le nom du certificat.
    * **publicKey** (*string*) - La clé publique du certificat.
    * **privateKey** (*string*) - La clé privée du certificat.
    * **certificat** (*string*) - Le certificat utilisateur.

  * **accounts** (*array*) - La liste des comptes rattaché au serveur.

    * **login** (*string*) - Le login du compte.
    * **password** (*string*) - Le mot de passe du compte. Si le compte a été
      exporté depuis une gateway existante, il s'agira d'un hash du mot de
      passe.
    * **certificates** (*array*) - La liste des :term:`certificats<certificat>`
      du compte.

      * **name** (*string*) - Le nom du certificat.
      * **publicKey** (*string*) - La clé publique du certificat.
      * **privateKey** (*string*) - La clé privée du certificat.
      * **certificat** (*string*) - Le certificat utilisateur.


* **remotes** (*array*) - La liste des :term:`partenaires<partenaire>` de
  transfert de la gateway.

  * **name** (*string*) - Le nom du partenaire.
  * **protocol** (*string*) - Le protocole du partenaire.
  * **configuration** (*object*) - La :any:`configuration protocolaire
    <reference-proto-config>` du serveur.
  * **certificates** (*array*) - La liste des :term:`certificats
    <certificat>` du partenaire.

    * **name** (*string*) - Le nom du certificat.
    * **publicKey** (*string*) - La clé publique du certificat.
    * **privateKey** (*string*) - La clé privée du certificat.
    * **certificat** (*string*) - Le certificat utilisateur.

  * **accounts** (*array*) - La liste des comptes rattaché au partenaire.

    * **login** (*string*) - Le login du compte.
    * **password** (*string*) - Le mot de passe du compte.
    * **certificates** (*array*) - La liste des :term:`certificats<certificat>`
      du compte.

      * **name** (*string*) - Le nom du certificat.
      * **publicKey** (*string*) - La clé publique du certificat.
      * **privateKey** (*string*) - La clé privée du certificat.
      * **certificat** (*string*) - Le certificat utilisateur.


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


**Exemple**

.. code-block:: json

   {
     "locals": [{
       "name": "serveur_sftp",
       "protocol": "sftp",
       "root": "/sftp",
       "workDir": "/sftp/tmp",
       "configuration": {
         "address": "localhost",
         "port": 8022
       },
       "accounts": [{
         "login": "toto",
         "password": "sésame",
         "certs": [{
           "name": "cert_toto",
           "publicKey": "<clé publique>",
           "privateKey": "<clé privée>",
           "certificate": "<certificat>"
         }]
       }],
       "certs": [{
         "name": "cert_serveur_sftp",
         "publicKey": "<clé publique>",
         "privateKey": "<clé privée>",
         "certificate": "<certificat>"
       }]
     }],
     "remotes": [{
       "name": "openssh",
       "protocol": "sftp",
       "configuration": {
         "address": "localhost",
         "port": 22
       },
       "accounts": [{
         "login": "titi",
         "password": "sésame",
         "certs": [{
           "name": "cert_titi",
           "publicKey": "<clé publique>",
           "privateKey": "<clé privée>",
           "certificate": "<certificat>"
         }]
       }],
       "certs": [{
         "name": "cert_openssh",
         "publicKey": "<clé publique>",
         "privateKey": "<clé privée>",
         "certificate": "<certificat>"
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
