Redémarrer un client
====================

.. http:delete:: /api/clients/(string:client_name)/restart

   Arrête et relance le client demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le client a été redémarré avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le client demandé n'existe pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/clients/sftp_client/restart HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED