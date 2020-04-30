Ajouter un certificat
=====================

.. http:post:: /api/partners/(string:partner)/certificates

   Ajoute un nouveau certificat à partir des informations renseignées en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string privateKey: La clé privée du certificat
   :reqjson string publicKey: La clé publique du certificat
   :reqjson string certificate: Le certificat de l'entité

   :statuscode 201: Le certificat a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du certificat sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au nouveau certificat créé


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/partners/waarp_sftp/certificates HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 156

         {
           "name": "certificat_waarp",
           "privateKey": "<clé privée>",
           "publicKey": "<clé publique>",
           "cert": "<certificat>"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/partners/waarp_sftp/certificates/certificat_waarp