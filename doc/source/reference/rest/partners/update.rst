Modifier un partenaire
======================

.. http:patch:: /api/partners/(string:partner_name)

   Met à jour le partenaire demandé avec les informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du partenaire
   :reqjson string protocol: Le protocole utilisé par le partenaire
   :reqjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.

   :statuscode 201: Le partenaire a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au partenaire modifié


   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/partners/waarp_sftp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 148

         {
           "name": "waarp_sftp_new",
           "protocol": "sftp",
           "protoConfig": {
             "address": "waarp.org",
             "port": 23
           }
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/partners/waarp_sftp_new