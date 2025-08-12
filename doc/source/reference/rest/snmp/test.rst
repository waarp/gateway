Envoyer une notification de test
================================

.. http:put:: /api/snmp/test/trap

   Envoie une notification (*trap*) SNMP de test à tous les moniteurs SNMP déjà
   configurés. Cette notification imitera les notifications produites en cas
   d'erreur de transfert.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 202: La notification a été envoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide


   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/snmp/test/trap HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 202 ACCEPTED
