Supprimer un utilisateur
========================

.. http:delete:: /api/users/(string:username)

   Supprime l'utilisateur demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: L'utilisateur a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/users/toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT