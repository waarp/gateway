Créer un serveur
================

.. http:post:: /api/servers

   Ajoute un nouveau serveur avec les informations renseignées en JSON.

   .. warning:: Les dossiers d'envoi, de réception et de travail devant rester
      distincts, une valeur par défaut leur sera attribuée si l'utilisateur
      renseigne une racine (``root``) sans donner de valeur aux sous-dossiers.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson string address: L'adresse du serveur (en format [adresse:port])
   :reqjson string root: La racine du serveur. Peut être relatif (à la racine
      de la *gateway*) ou absolu (OBSOLÈTE: remplacé par 'rootDir').
   :reqjson string inDir: Le dossier de réception du serveur. Peut être
      relatif (à la racine du serveur) ou absolu. (OBSOLÈTE: remplacé par
      'receiveDir')
   :reqjson string outDir: Le dossier d'envoi du serveur. Peut être
      relatif (à la racine du serveur) ou absolu. (OBSOLÈTE: remplacé par
      'sendDir')
   :reqjson string workDir: Le dossier temporaire du serveur. Peut être
      relatif (à la racine du serveur) ou absolu. (OBSOLÈTE: remplacé par
      'tmpReceiveDir')
   :reqjson string rootDir: Chemin du dossier racine du serveur. Peut être
      relatif (à la racine de la *gateway*) ou absolu.
   :reqjson string receiveDir: Le dossier de réception du serveur. Peut
      être relatif (à la racine du serveur) ou absolu.
   :reqjson string sendDir: Le dossier d'envoi du serveur. Peut être
      relatif (à la racine du serveur) ou absolu.
   :reqjson string tmpReceiveDir: Le dossier temporaire du serveur. Peut
      être relatif (à la racine du serveur) ou absolu.
   :reqjson object protoConfig: La configuration du serveur encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.

   :statuscode 201: Le serveur a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau serveur créé


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/servers HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 140

         {
           "name": "sftp_server",
           "protocol": "sftp",
           "address": "localhost:2022",
           "rootDir": "/sftp/root",
           "protoConfig": {}
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/sftp_server
