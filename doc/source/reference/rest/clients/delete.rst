Supprimer un client
===================
.. http:delete:: /api/clients/(string:client_name)

   Supprime le client demandé. Nécessite le droit de suppression sur les
   serveurs.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le client a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le client demandé n'existe pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/clients/sftp_client HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT