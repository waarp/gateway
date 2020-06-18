Supprimer un compte distant
===========================

.. http:delete:: /api/partners/(string:partner_name)/accounts/(string:login)

   Supprime le compte ``login`` du partenaire donné.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le compte a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/partner/waarp_sftp/account/titi HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT