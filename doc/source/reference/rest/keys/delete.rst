Supprimer une clé cryptographique
=================================

.. http:delete:: /api/keys/(string:key_name)

   Supprime la clé cryptographique demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: La clé a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La clé demandée n'existe pas

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/keys/aes-key HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO-CONTENT
