Créer un partenaire
===================

.. http:post:: /api/partners

   Ajoute un nouveau partenaire avec les informations renseignées en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du partenaire
   :reqjson string protocol: Le protocole utilisé par le partenaire
   :reqjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.

   :statuscode 201: Le partenaire a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau partenaire créé


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/partners HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 140

         {
           "name": "waarp_sftp",
           "protocol": "sftp",
           "protoConfig": {
             "address": "waarp.org",
             "port": 21
           }
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/partners/waarp_sftp