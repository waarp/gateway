Modifier un serveur
===================

.. http:put:: /api/servers/(string:server_name)

   Met à jour le serveur demandé avec les informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson string root: Le dossier racine du serveur
   :reqjson object protoConfig: La configuration du serveur encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.

   :statuscode 201: Le serveur a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas

   :resheader Location: Le chemin d'accès au serveur modifié


   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/servers/sftp_server HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 148

         {
           "name": "sftp_server_new",
           "protocol": "sftp",
           "root": "/new/sftp/root",
           "protoConfig": {
             "address": "localhost",
             "port": 23
           }
         }

   |

   **Exemple de requête**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/sftp_server_new