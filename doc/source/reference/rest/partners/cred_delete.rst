Supprimer une valeur d'authentification
=======================================

.. http:delete:: /api/partners/(string:partner_name)/credentials/(string:auth_value_name)

   Supprime la valeur d'authentification donnée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: La valeur a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/partners/openssh/credentials/openssh_hostkey HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT