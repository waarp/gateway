Consulter un utilisateur
========================

.. http:get:: /api/users/(string:username)

   Renvoie l'utilisateur demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: L'utilisateur a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   :resjson number id: L'identifiant unique de l'utilisateur
   :resjson string username: Le nom de l'utilisateur


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/users/toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 41

         {
           "username": "toto"
         }