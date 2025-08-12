Supprimer un identifiant SMTP
=============================

.. http:delete:: /api/email/credentials/(string:name)

   Supprime l'identifiant SMTP demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 204: L'identifiant a été supprimé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: L'identifiant demandé n'existe pas.


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/email/credentials/waarp@example.com HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
