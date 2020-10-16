Remplacer un utilisateur
========================

.. http:put:: /api/users/(string:username)

   Remplace l'utilisateur demandé par celui renseigné en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string username: Le nom de l'utilisateur
   :reqjson string password: Le mot de passe de l'utilisateur

   :statuscode 201: L'utilisateur a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres de l'utilisateur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   :resheader Location: Le chemin d'accès à l'utilisateur modifié


   |

   **Exemple de requête**

      .. code-block:: http

         PUT https://my_waarp_gateway.net/api/users/toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 84

         {
           "username": "toto_new",
           "password": "titi_new"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/users/toto_new