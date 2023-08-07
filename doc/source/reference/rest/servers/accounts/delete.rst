Supprimer un compte local
=========================

.. http:delete:: /api/servers/(string:server_name)/accounts/(string:login)

   Supprime le compte ``login`` du serveur donné.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le compte a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/server/sftp_server/account/toto HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
