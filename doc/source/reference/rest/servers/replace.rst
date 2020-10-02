Remplacer un serveur
====================

.. http:put:: /api/servers/(string:server_name)

   Remplace le serveur demandé par celui renseigné en JSON.

   .. warning:: Les dossiers d'envoi, de réception et de travail devant rester
      distincts, une valeur par défaut leur sera attribuée si l'utilisateur renseigne
      une racine (``root``) sans donner de valeur aux sous-dossiers.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson string address: L'adresse du serveur (en format [adresse:port])
   :reqjson string root: La racine du serveur. Peut être relatif (à la racine
      de la *gateway*) ou absolu.
   :reqjson string inDir: Le dossier de réception du serveur. Peut être
      relatif (à la racine du serveur) ou absolu.
   :reqjson string outDir: Le dossier d'envoi du serveur. Peut être
      relatif (à la racine du serveur) ou absolu.
   :reqjson string workDir: Le dossier temporaire du serveur. Peut être
      relatif (à la racine du serveur) ou absolu.
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

         PUT https://my_waarp_gateway.net/api/servers/sftp_server HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 148

         {
           "name": "sftp_server_new",
           "protocol": "sftp",
           "address": "localhost:2023",
           "root": "/new/sftp/root",
           "protoConfig": {}
         }

   |

   **Exemple de requête**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/sftp_server_new