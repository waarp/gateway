Remplacer un certificat
=======================

.. http:put:: /api/partners/(string:partner)/certificates/(string:cert_name)

   Remplace le certificat demandé par celui renseigné en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string privateKey: La clé privée du certificat
   :reqjson string publicKey: La clé publique du certificat
   :reqjson string certificate: Le certificat de l'entité

   :statuscode 201: Le certificat a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou le certificat demandés n'existent pas

   :resheader Location: Le chemin d'accès au certificat modifié


   |

   **Exemple de requête**

      .. code-block:: http

         PUT https://my_waarp_gateway.net/api/partners/waarp_sftp/certificates/certificat_waarp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 69

         {
           "name": "certificat_waarp_new",
           "privateKey": "<clé privée>",
           "publicKey": "<clé publique>",
           "cert": "<certificat>"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/partners/waarp_sftp/certificates/certtificat_sftp