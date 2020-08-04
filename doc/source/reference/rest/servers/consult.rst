Consulter un serveur
====================

.. http:get:: /api/servers/(string:server_name)

   Renvoie les informations du serveur portant le nom ``server_name``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du serveur ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas

   :resjson string name: Le nom du serveur
   :resjson string protocol: Le protocole utilisé par le serveur
   :resjson object paths: Les différents dossiers du serveur.

      * **root** (*string*) - La racine du serveur. Peut être relatif (à la racine
        de la *gateway*) ou absolu.
      * **inDir** (*string*) - Le dossier de réception du serveur. Peut être
        relatif (à la racine du serveur) ou absolu.
      * **outDir** (*string*) - Le dossier d'envoi du serveur. Peut être
        relatif (à la racine du serveur) ou absolu.
      * **workDir** (*string*) - Le dossier temporaire du serveur. Peut être
        relatif (à la racine du serveur) ou absolu.

   :resjson object protoConfig: La configuration du serveur encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjson object authorizedRules: Les règles que le serveur est autorisé à
      utiliser pour les transferts.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/servers/sftp_server HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 271

         {
           "name": "sftp_server",
           "protocol": "sftp",
           "root": "/sftp/root",
           "protoConfig": {
             "address": "localhost",
             "port": 21
           },
           "authorizedRules": {
             "sending": ["règle_envoi_1", "règle_envoi_2"],
             "reception": ["règle_récep_1", "règle_récep_2"]
           }
         }