Supprimer un serveur
====================

.. http:delete:: /api/servers/(string:server_name)

   Supprime le serveur demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le serveur a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/servers/sftp_server HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
