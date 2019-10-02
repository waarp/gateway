Supprimer un certificat
=======================

.. http:delete:: /api/certificates/(int:certificate_id)

   Supprime le certificat portant le numéro ``certificate_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          DELETE https://my_waarp_gateway.net/api/certificates/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: Le certificat a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat demandé n'existe pas

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 204 NO CONTENT